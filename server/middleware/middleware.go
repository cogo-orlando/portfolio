package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════
//  LOGGING
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
		log.Printf("[%s] %s %s %d %dms %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method, r.URL.Path, rw.status,
			time.Since(start).Milliseconds(),
			GetIP(r),
		)
	})
}

func GetIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
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
//  RATE LIMITER
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

func init() {
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
		if !globalRateLimiter.allow(GetIP(r)) {
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
//  HEALTH — exporté, sans import server
// ══════════════════════════════════════════

var startTime = time.Now()

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","service":"portfolio","uptime":"%s"}`,
		time.Since(startTime).Round(time.Second).String(),
	)
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
