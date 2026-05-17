package middleware

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"portfo/server/db"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ══════════════════════════════════════════
//  INIT SLOG
// ══════════════════════════════════════════

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

// ══════════════════════════════════════════
//  REQUEST ID
// ══════════════════════════════════════════

type contextKey string

const RequestIDKey contextKey = "request_id"

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := generateRequestID()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "-"
}

// ══════════════════════════════════════════
//  PANIC RECOVERY
// ══════════════════════════════════════════

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				id := GetRequestID(r.Context())
				slog.Error("panic récupéré",
					"request_id", id,
					"path", r.URL.Path,
					"error", fmt.Sprintf("%v", err),
					"stack", string(debug.Stack()),
				)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "500 — Erreur interne du serveur")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  CIRCUIT BREAKER — DB
//  Si la DB échoue 5 fois → ouvre le circuit
//  pendant 30 secondes avant de réessayer
// ══════════════════════════════════════════

type circuitState int32

const (
	circuitClosed   circuitState = 0
	circuitOpen     circuitState = 1
	circuitHalfOpen circuitState = 2
)

type dbCircuitBreaker struct {
	state        atomic.Int32
	failures     atomic.Int32
	lastFailure  atomic.Int64
	maxFailures  int32
	resetTimeout time.Duration
}

var DBCircuit = &dbCircuitBreaker{
	maxFailures:  5,
	resetTimeout: 30 * time.Second,
}

func (cb *dbCircuitBreaker) Allow() bool {
	state := circuitState(cb.state.Load())
	switch state {
	case circuitClosed:
		return true
	case circuitOpen:
		last := time.Unix(0, cb.lastFailure.Load())
		if time.Since(last) >= cb.resetTimeout {
			cb.state.Store(int32(circuitHalfOpen))
			return true
		}
		return false
	default:
		return true
	}
}

func (cb *dbCircuitBreaker) RecordSuccess() {
	cb.failures.Store(0)
	cb.state.Store(int32(circuitClosed))
}

func (cb *dbCircuitBreaker) RecordFailure() {
	failures := cb.failures.Add(1)
	cb.lastFailure.Store(time.Now().UnixNano())
	if failures >= cb.maxFailures {
		if cb.state.Load() != int32(circuitOpen) {
			slog.Warn("circuit breaker DB ouvert — logs DB suspendus 30s",
				"failures", failures,
			)
		}
		cb.state.Store(int32(circuitOpen))
	}
}

func (cb *dbCircuitBreaker) IsOpen() bool {
	return circuitState(cb.state.Load()) == circuitOpen
}

// ══════════════════════════════════════════
//  HONEYPOT
// ══════════════════════════════════════════

var honeypotRoutes = map[string]bool{
	"/admin/login":   true,
	"/admin":         true,
	"/wp-admin":      true,
	"/wp-login.php":  true,
	"/wp-config.php": true,
	"/.env":          true,
	"/config.php":    true,
	"/phpinfo.php":   true,
	"/.git/config":   true,
	"/etc/passwd":    true,
	"/api/admin":     true,
	"/administrator": true,
	"/login":         true,
	"/shell":         true,
	"/console":       true,
}

type blacklist struct {
	mu      sync.RWMutex
	records map[string]time.Time
}

var ipBlacklist = &blacklist{records: make(map[string]time.Time)}

func (bl *blacklist) add(ip string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.records[ip] = time.Now().Add(24 * time.Hour)
}

func (bl *blacklist) has(ip string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	exp, ok := bl.records[ip]
	return ok && time.Now().Before(exp)
}

func init() { //nolint:gochecknoinits
	go func() {
		for range time.Tick(time.Hour) {
			ipBlacklist.mu.Lock()
			now := time.Now()
			for ip, exp := range ipBlacklist.records {
				if now.After(exp) {
					delete(ipBlacklist.records, ip)
				}
			}
			ipBlacklist.mu.Unlock()
		}
	}()
}

func HoneypotMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetIP(r)
		if ipBlacklist.has(ip) {
			slog.Warn("ip blacklistée bloquée",
				"ip", ip,
				"path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if honeypotRoutes[r.URL.Path] {
			ipBlacklist.add(ip)
			slog.Warn("honeypot déclenché — IP blacklistée 24h",
				"ip", ip,
				"path", r.URL.Path,
				"method", r.Method,
				"user_agent", r.UserAgent(),
				"request_id", GetRequestID(r.Context()),
			)
			if DBCircuit.Allow() {
				db.LogHoneypot(ip, r.URL.Path, r.UserAgent())
			}
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  LOGGING MIDDLEWARE
// ══════════════════════════════════════════

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		ms := time.Since(start).Milliseconds()
		ip := GetIP(r)
		id := GetRequestID(r.Context())

		switch {
		case rw.status >= 500:
			slog.Error("request",
				"request_id", id, "method", r.Method, "path", r.URL.Path,
				"status", rw.status, "duration_ms", ms, "ip", ip,
			)
		case rw.status >= 400:
			slog.Warn("request",
				"request_id", id, "method", r.Method, "path", r.URL.Path,
				"status", rw.status, "duration_ms", ms, "ip", ip,
			)
		default:
			slog.Info("request",
				"request_id", id, "method", r.Method, "path", r.URL.Path,
				"status", rw.status, "duration_ms", ms, "ip", ip,
			)
		}

		if DBCircuit.Allow() {
			eventType := db.EventRequest
			if rw.status >= 400 {
				eventType = db.EventError
			}
			db.LogEvent(ip, r.Method, r.URL.Path, rw.status, r.UserAgent(), eventType)
		}
	})
}

// ══════════════════════════════════════════
//  IP RÉELLE (Cloudflare en priorité)
// ══════════════════════════════════════════

func GetIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	return r.RemoteAddr
}

// ══════════════════════════════════════════
//  SECURITY
// ══════════════════════════════════════════

func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Del("X-Powered-By")
		h.Set("Server", "")
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  GZIP (Brotli nécessite lib externe)
// ══════════════════════════════════════════

var skipCompressExts = []string{
	".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".woff", ".woff2",
}

func shouldSkipCompression(path string) bool {
	for _, ext := range skipCompressExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (gw *gzipWriter) Write(b []byte) (int, error) { return gw.gz.Write(b) }
func (gw *gzipWriter) WriteHeader(status int) {
	gw.ResponseWriter.Header().Del("Content-Length")
	gw.ResponseWriter.WriteHeader(status)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldSkipCompression(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&gzipWriter{ResponseWriter: w, gz: gz}, r)
	})
}

// ══════════════════════════════════════════
//  CACHE
// ══════════════════════════════════════════

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/img/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(r.URL.Path, "/css/") || strings.HasPrefix(r.URL.Path, "/js/"):
			w.Header().Set("Cache-Control", "public, max-age=604800")
		default:
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  RATE LIMITER (120 req/min/IP)
// ══════════════════════════════════════════

type globalLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

var globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}

func (gl *globalLimiter) allow(ip string) bool {
	gl.mu.Lock()
	defer gl.mu.Unlock()
	now := time.Now()
	var recent []time.Time
	for _, t := range gl.records[ip] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 120 {
		return false
	}
	gl.records[ip] = append(recent, now)
	return true
}

func init() { //nolint:gochecknoinits
	go func() {
		for range time.Tick(5 * time.Minute) {
			globalRateLimiter.mu.Lock()
			now := time.Now()
			for ip, times := range globalRateLimiter.records {
				var recent []time.Time
				for _, t := range times {
					if now.Sub(t) < time.Minute {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(globalRateLimiter.records, ip)
				} else {
					globalRateLimiter.records[ip] = recent
				}
			}
			globalRateLimiter.mu.Unlock()
		}
	}()
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetIP(r)
		if !globalRateLimiter.allow(ip) {
			slog.Warn("rate limit global dépassé",
				"ip", ip, "path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			if DBCircuit.Allow() {
				db.LogRateLimit(ip, r.URL.Path)
			}
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  TIMEOUT PAR ROUTE
//  API/health  : 3s
//  /projects/  : 8s
//  autres      : 5s
// ══════════════════════════════════════════

func timeoutForPath(path string) time.Duration {
	switch {
	case strings.HasPrefix(path, "/api/") || path == "/health":
		return 3 * time.Second
	case strings.HasPrefix(path, "/projects/"):
		return 8 * time.Second
	default:
		return 5 * time.Second
	}
}

func TimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeout := timeoutForPath(r.URL.Path)
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ══════════════════════════════════════════
//  CHAIN
// ══════════════════════════════════════════

func Chain(h http.Handler) http.Handler {
	h = CacheMiddleware(h)
	h = GzipMiddleware(h)
	h = SecurityMiddleware(h)
	h = HoneypotMiddleware(h)
	h = RateLimitMiddleware(h)
	h = TimeoutMiddleware(h)
	h = RecoveryMiddleware(h)
	h = RequestIDMiddleware(h)
	h = LoggerMiddleware(h)
	return h
}

// ══════════════════════════════════════════
//  HEALTH
// ══════════════════════════════════════════

var startTime = time.Now()

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","service":"portfolio","uptime":"%s"}`,
		time.Since(startTime).Round(time.Second).String(),
	)
}

// ══════════════════════════════════════════
//  CONTEXT LOGGER
// ══════════════════════════════════════════

const LoggerKey contextKey = "logger"

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ══════════════════════════════════════════
//  TIMEOUT WRITER
// ══════════════════════════════════════════

type timeoutWriter struct {
	http.ResponseWriter
	mu   sync.Mutex
	done bool
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.done {
		return 0, io.ErrClosedPipe
	}
	return tw.ResponseWriter.Write(b)
}
