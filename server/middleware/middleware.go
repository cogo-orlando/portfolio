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
	"runtime/debug"
	"strings"
	"sync"
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
//  Chaque requête reçoit un ID unique
//  visible dans tous les logs
// ══════════════════════════════════════════

type contextKey string

const RequestIDKey contextKey = "request_id"

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
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
//  Si une route panic, renvoie un 500 propre
//  sans crasher le serveur
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
//  HONEYPOT
//  Routes fausses qui attirent les bots/scanners
//  Toute IP qui y accède est loguée comme suspecte
// ══════════════════════════════════════════

// Liste des routes honeypot — jamais visitées par un humain normal
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

// Blacklist en mémoire des IPs suspectes
type blacklist struct {
	mu      sync.RWMutex
	records map[string]time.Time // ip → expiration
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

// Nettoyage auto toutes les heures
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

		// Bloque les IPs déjà blacklistées
		if ipBlacklist.has(ip) {
			slog.Warn("ip blacklistée bloquée",
				"ip", ip,
				"path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Vérifie si la route est un honeypot
		if honeypotRoutes[r.URL.Path] {
			ipBlacklist.add(ip)
			slog.Warn("honeypot déclenché — IP blacklistée 24h",
				"ip", ip,
				"path", r.URL.Path,
				"method", r.Method,
				"user_agent", r.UserAgent(),
				"request_id", GetRequestID(r.Context()),
			)
			// Répond comme si la route n'existait pas
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
				"request_id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", ms,
				"ip", ip,
			)
		case rw.status >= 400:
			slog.Warn("request",
				"request_id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", ms,
				"ip", ip,
			)
		default:
			slog.Info("request",
				"request_id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", ms,
				"ip", ip,
			)
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
//  GZIP
// ══════════════════════════════════════════

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
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp"} {
			if strings.HasSuffix(r.URL.Path, ext) {
				next.ServeHTTP(w, r)
				return
			}
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
				"ip", ip,
				"path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
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
