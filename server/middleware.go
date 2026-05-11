package server

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
		duration := time.Since(start)
		ip := getIP(r)
		log.Printf("[%s] %s %s %d %dms %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			rw.status,
			duration.Milliseconds(),
			ip,
		)
	})
}

func getIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// ══════════════════════════════════════════
//  SECURITY HEADERS MIDDLEWARE
// ══════════════════════════════════════════

func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Force HTTPS
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Empêche le MIME sniffing
		h.Set("X-Content-Type-Options", "nosniff")

		// Empêche le clickjacking
		h.Set("X-Frame-Options", "DENY")

		// XSS protection basique
		h.Set("X-XSS-Protection", "1; mode=block")

		// Contrôle les informations de référence
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Désactive les fonctionnalités navigateur non nécessaires
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// Content Security Policy
		h.Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline'",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
			"font-src 'self' https://fonts.gstatic.com",
			"img-src 'self' data: https:",
			"connect-src 'self' https://api.open-meteo.com https://formspree.io",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self' https://formspree.io",
		}, "; "))

		// Supprime le header qui expose Go
		h.Del("X-Powered-By")
		w.Header().Set("Server", "")

		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  GZIP MIDDLEWARE
// ══════════════════════════════════════════

type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (gw *gzipWriter) Write(b []byte) (int, error) {
	return gw.gz.Write(b)
}

func (gw *gzipWriter) WriteHeader(status int) {
	gw.ResponseWriter.Header().Del("Content-Length")
	gw.ResponseWriter.WriteHeader(status)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ne compresse pas si le client ne supporte pas
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Ne compresse pas les images (déjà compressées)
		path := r.URL.Path
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".webp") {
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
//  CACHE MIDDLEWARE (fichiers statiques)
// ══════════════════════════════════════════

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// Images — 1 an
		case strings.HasPrefix(path, "/img/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		// CSS et JS — 1 semaine (peut changer lors des déploiements)
		case strings.HasPrefix(path, "/css/") || strings.HasPrefix(path, "/js/"):
			w.Header().Set("Cache-Control", "public, max-age=604800")

		// Pages HTML — pas de cache (contenu dynamique)
		default:
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  TIMEOUT MIDDLEWARE
// ══════════════════════════════════════════

func TimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Timeout de 30 secondes par requête
		done := make(chan struct{})
		go func() {
			next.ServeHTTP(w, r)
			close(done)
		}()
		select {
		case <-done:
			return
		case <-time.After(30 * time.Second):
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		}
	})
}

// ══════════════════════════════════════════
//  RATE LIMITER GLOBAL (par IP)
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
	window := time.Minute

	var recent []time.Time
	for _, t := range gl.records[ip] {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}

	// Max 120 requêtes par minute par IP
	if len(recent) >= 120 {
		return false
	}

	gl.records[ip] = append(recent, now)
	return true
}

// Nettoie les vieilles entrées toutes les 5 minutes
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
		ip := getIP(r)
		if !globalRateLimiter.allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  REDIRECT HTTP → HTTPS
// ══════════════════════════════════════════

func RedirectHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sur Render, X-Forwarded-Proto indique si la requête est HTTP ou HTTPS
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "http" {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  CHAIN — Applique tous les middlewares
// ══════════════════════════════════════════

func Chain(h http.Handler) http.Handler {
	// Ordre d'application (du dernier au premier)
	h = CacheMiddleware(h)
	h = GzipMiddleware(h)
	h = SecurityMiddleware(h)
	h = RateLimitMiddleware(h)
	h = RedirectHTTPS(h)
	h = LoggerMiddleware(h)
	return h
}

// ══════════════════════════════════════════
//  HEALTH CHECK ENRICHI
// ══════════════════════════════════════════

func healthHandler(w http.ResponseWriter, r *http.Request) {
	visitMu.Lock()
	visits := visitCount
	visitMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","service":"portfolio","visits":%d,"uptime":"%s"}`,
		visits,
		time.Since(startTime).Round(time.Second).String(),
	)
}

var startTime = time.Now()

// ══════════════════════════════════════════
//  WRITER HELPER pour TimeoutMiddleware
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
