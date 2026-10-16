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
				writeError(w, r, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
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
		if ipBlacklist.has(ip) && !isStaticAsset(r.URL.Path) && r.URL.Path != "/health" {
			slog.Warn("ip blacklistée bloquée",
				"ip", ip,
				"path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			writeError(w, r, http.StatusForbidden)
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
			writeNotFound(w, r) // même 404 que partout ailleurs : le piège reste invisible
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
		// Journal des visites : IP anonymisée (RGPD). Les alertes de sécurité
		// (honeypot, rate limit, blacklist) gardent l'IP complète.
		ip := AnonymizeIP(GetIP(r))
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
	})
}

// IP réelle du visiteur : voir clientip.go (GetIP, CloudflareOnlyMiddleware)

// ══════════════════════════════════════════
//  SECURITY
//  Content-Security-Policy stricte : AUCUN script inline n'est exécuté.
//  Les seules sources externes autorisées sont celles réellement utilisées.
// ══════════════════════════════════════════

// ContentSecurityPolicy est exportée pour être vérifiée par les tests
const ContentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " + // uniquement nos fichiers .js : pas d'inline, pas d'eval
	"style-src 'self' 'unsafe-inline'; " + // attributs style="--var:…" utilisés pour les couleurs
	"img-src 'self' data:; " +
	"font-src 'self'; " + // polices hébergées sur le site
	"connect-src 'self' https://api.open-meteo.com https://formspree.io; " +
	"form-action 'self' https://formspree.io; " +
	"frame-src 'self'; " +
	"frame-ancestors 'none'; " + // le site ne peut pas être intégré dans une iframe (clickjacking)
	"base-uri 'self'; " +
	"object-src 'none'; " +
	"upgrade-insecure-requests"

// setSecurityHeaders pose les en-têtes de sécurité (aussi utilisé par les pages d'erreur)
func setSecurityHeaders(h http.Header) {
	h.Del("X-Powered-By")
	h.Set("Server", "")
	h.Set("Content-Security-Policy", ContentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w.Header())
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
//  - /img/ et /fonts/   : 1 an (contenu qui ne change pas)
//  - CSS/JS versionnés  : 1 an, immutable → l'URL change à chaque modif (?v=hash)
//  - CSS/JS sans ?v=    : 1 heure, par sécurité
//  - HTML               : toujours revalidé (l'ETag renvoie un 304 si rien n'a changé)
// ══════════════════════════════════════════

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		isAsset := strings.HasPrefix(path, "/css/") || strings.HasPrefix(path, "/js/")

		switch {
		case strings.HasPrefix(path, "/img/") || strings.HasPrefix(path, "/fonts/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case isAsset && r.URL.Query().Get("v") != "":
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case isAsset:
			w.Header().Set("Cache-Control", "public, max-age=3600")
		default:
			w.Header().Set("Cache-Control", "no-cache")
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
		if isStaticAsset(r.URL.Path) { // voir isStaticAsset (errors.go)
			next.ServeHTTP(w, r)
			return
		}
		ip := GetIP(r)
		if !globalRateLimiter.allow(ip) {
			slog.Warn("rate limit global dépassé",
				"ip", ip, "path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
			w.Header().Set("Retry-After", "60")
			writeError(w, r, http.StatusTooManyRequests)
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
//  Ordre d'exécution (de l'extérieur vers l'intérieur) :
//  RequestID → Logger → CloudflareOnly → Recovery → Maintenance → Timeout
//  → RateLimit → Honeypot → Security → Gzip → Cache → routes
// ══════════════════════════════════════════

func Chain(h http.Handler) http.Handler {
	h = CacheMiddleware(h)
	h = GzipMiddleware(h)
	h = SecurityMiddleware(h)
	h = HoneypotMiddleware(h)
	h = RateLimitMiddleware(h)
	h = TimeoutMiddleware(h)
	h = MaintenanceMiddleware(h)
	h = RecoveryMiddleware(h)
	h = CloudflareOnlyMiddleware(h)
	h = LoggerMiddleware(h)
	h = RequestIDMiddleware(h) // en dernier = s'exécute en premier
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
