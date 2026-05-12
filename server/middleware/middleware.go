package middleware

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════
//  INIT SLOG — JSON structuré vers stdout
//  Lisible directement dans Render Logs
// ══════════════════════════════════════════

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
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

		// Niveau de log selon le status HTTP
		switch {
		case rw.status >= 500:
			slog.Error("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", ms,
				"ip", ip,
			)
		case rw.status >= 400:
			slog.Warn("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", ms,
				"ip", ip,
			)
		default:
			slog.Info("request",
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
			slog.Warn("rate limit global dépassé", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  REDIRECT HTTPS
// ══════════════════════════════════════════

func RedirectHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "http" {
			http.Redirect(w, r, "https://"+r.Host+r.URL.RequestURI(), http.StatusMovedPermanently)
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
	h = RateLimitMiddleware(h)
	h = RedirectHTTPS(h)
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
//  CONTEXT KEY (pour passer le logger aux handlers)
// ══════════════════════════════════════════

type contextKey string

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
