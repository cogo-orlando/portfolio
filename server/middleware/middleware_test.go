package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════
//  HELPERS
// ══════════════════════════════════════════

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func newRequest(method, path, ip string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("CF-Connecting-IP", ip)
	return r
}

// ══════════════════════════════════════════
//  TESTS — GetIP
// ══════════════════════════════════════════

func TestGetIP_CloudflareHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	ip := GetIP(r)
	if ip != "1.2.3.4" {
		t.Errorf("attendu 1.2.3.4, obtenu %s", ip)
	}
}

func TestGetIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "5.6.7.8, 9.9.9.9")
	ip := GetIP(r)
	if ip != "5.6.7.8" {
		t.Errorf("attendu 5.6.7.8, obtenu %s", ip)
	}
}

func TestGetIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:1234"
	ip := GetIP(r)
	if ip != "192.168.1.1:1234" {
		t.Errorf("attendu 192.168.1.1:1234, obtenu %s", ip)
	}
}

func TestGetIP_CloudflarePriority(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.1.1.1")
	r.Header.Set("X-Forwarded-For", "2.2.2.2")
	r.RemoteAddr = "3.3.3.3:80"
	ip := GetIP(r)
	if ip != "1.1.1.1" {
		t.Errorf("CF-Connecting-IP devrait avoir la priorité, obtenu %s", ip)
	}
}

// ══════════════════════════════════════════
//  TESTS — RateLimitMiddleware
// ══════════════════════════════════════════

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	handler := RateLimitMiddleware(okHandler())
	ip := "10.0.0.1"
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r := newRequest("GET", "/", ip)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("requête %d : attendu 200, obtenu %d", i+1, w.Code)
		}
	}
}

func TestRateLimit_BlocksAfterLimit(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	ip := "10.0.0.2"
	now := time.Now()
	times := make([]time.Time, 120)
	for i := range times {
		times[i] = now
	}
	globalRateLimiter.records[ip] = times

	handler := RateLimitMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := newRequest("GET", "/", ip)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("attendu 429, obtenu %d", w.Code)
	}
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	ip := "10.0.0.6"
	now := time.Now()
	times := make([]time.Time, 120)
	for i := range times {
		times[i] = now
	}
	globalRateLimiter.records[ip] = times

	handler := RateLimitMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := newRequest("GET", "/", ip)
	handler.ServeHTTP(w, r)

	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header devrait être présent sur 429")
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	handler := RateLimitMiddleware(okHandler())

	ipA := "10.0.1.1"
	now := time.Now()
	times := make([]time.Time, 120)
	for i := range times {
		times[i] = now
	}
	globalRateLimiter.records[ipA] = times

	ipB := "10.0.1.2"
	w := httptest.NewRecorder()
	r := newRequest("GET", "/", ipB)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("IP B ne devrait pas être bloquée, obtenu %d", w.Code)
	}
}

func TestRateLimit_ExpiredRequestsNotCounted(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	ip := "10.0.0.7"

	// 120 requêtes expirées (il y a 2 minutes)
	old := time.Now().Add(-2 * time.Minute)
	times := make([]time.Time, 120)
	for i := range times {
		times[i] = old
	}
	globalRateLimiter.records[ip] = times

	handler := RateLimitMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := newRequest("GET", "/", ip)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("requêtes expirées ne devraient pas compter, obtenu %d", w.Code)
	}
}

func TestRateLimit_AllowMethod(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	ip := "10.0.0.8"

	// 119 requêtes — une de moins que la limite
	now := time.Now()
	times := make([]time.Time, 119)
	for i := range times {
		times[i] = now
	}
	globalRateLimiter.records[ip] = times

	if !globalRateLimiter.allow(ip) {
		t.Error("119 requêtes devrait encore être autorisé")
	}
}

// ══════════════════════════════════════════
//  TESTS — HoneypotMiddleware
// ══════════════════════════════════════════

func TestHoneypot_NormalRoute(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	handler := HoneypotMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := newRequest("GET", "/home", "20.0.0.1")
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("/home devrait passer, obtenu %d", w.Code)
	}
}

func TestHoneypot_BlacklistsOnTrap(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	handler := HoneypotMiddleware(okHandler())
	ip := "20.0.0.2"

	w := httptest.NewRecorder()
	r := newRequest("GET", "/wp-admin", ip)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("honeypot devrait retourner 404, obtenu %d", w.Code)
	}
	if !ipBlacklist.has(ip) {
		t.Error("l'IP devrait être blacklistée après avoir touché le honeypot")
	}
}

func TestHoneypot_BlocksBlacklistedIP(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	ip := "20.0.0.3"
	ipBlacklist.add(ip)

	handler := HoneypotMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := newRequest("GET", "/home", ip)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("IP blacklistée devrait obtenir 403, obtenu %d", w.Code)
	}
}

func TestHoneypot_AllTraps(t *testing.T) {
	traps := []string{
		"/wp-admin", "/wp-login.php", "/.env", "/phpinfo.php",
		"/.git/config", "/admin/login", "/shell", "/console",
		"/admin", "/wp-config.php", "/config.php", "/etc/passwd",
		"/api/admin", "/administrator", "/login",
	}
	for _, path := range traps {
		ipBlacklist = &blacklist{records: make(map[string]time.Time)}
		handler := HoneypotMiddleware(okHandler())
		w := httptest.NewRecorder()
		r := newRequest("GET", path, "30.0.0.1")
		handler.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("route honeypot %s ne devrait pas retourner 200", path)
		}
	}
}

func TestHoneypot_BlacklistExpiry(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	ip := "20.0.0.9"

	// Ajoute une entrée déjà expirée
	ipBlacklist.mu.Lock()
	ipBlacklist.records[ip] = time.Now().Add(-1 * time.Hour)
	ipBlacklist.mu.Unlock()

	if ipBlacklist.has(ip) {
		t.Error("IP avec expiration passée ne devrait pas être blacklistée")
	}
}

func TestHoneypot_BlacklistAdd(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	ip := "20.0.0.10"

	ipBlacklist.add(ip)

	if !ipBlacklist.has(ip) {
		t.Error("IP ajoutée devrait être dans la blacklist")
	}
}

func TestHoneypot_BlacklistFresh(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	ip := "20.0.0.11"

	// IP non blacklistée
	if ipBlacklist.has(ip) {
		t.Error("IP non ajoutée ne devrait pas être dans la blacklist")
	}
}

func TestHoneypot_TrapThenBlocked(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
	handler := HoneypotMiddleware(okHandler())
	ip := "20.0.0.12"

	// 1. Déclenche le honeypot
	w1 := httptest.NewRecorder()
	r1 := newRequest("GET", "/wp-admin", ip)
	handler.ServeHTTP(w1, r1)

	// 2. Tente une route normale — doit être bloqué
	w2 := httptest.NewRecorder()
	r2 := newRequest("GET", "/home", ip)
	handler.ServeHTTP(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("après honeypot, /home devrait être 403, obtenu %d", w2.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — RecoveryMiddleware
// ══════════════════════════════════════════

func TestRecovery_NormalHandler(t *testing.T) {
	handler := RecoveryMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("attendu 200, obtenu %d", w.Code)
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	handler := RecoveryMiddleware(panicHandler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRecovery_CatchesNilPointer(t *testing.T) {
	nilHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s *string
		_ = *s
	})
	handler := RecoveryMiddleware(nilHandler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("nil pointer devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRecovery_BodyContainsMessage(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("crash")
	})
	handler := RecoveryMiddleware(panicHandler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "500") {
		t.Error("body devrait contenir '500' après un panic")
	}
}

// ══════════════════════════════════════════
//  TESTS — SecurityMiddleware
// ══════════════════════════════════════════

func TestSecurity_ServerHeaderEmpty(t *testing.T) {
	handler := SecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Go/1.23")
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("attendu 200, obtenu %d", w.Code)
	}
}

func TestSecurity_XPoweredByRemoved(t *testing.T) {
	handler := SecurityMiddleware(okHandler())
	w := httptest.NewRecorder()
	w.Header().Set("X-Powered-By", "Go") // positionné avant
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	if w.Header().Get("X-Powered-By") != "" {
		t.Error("X-Powered-By devrait être supprimé par SecurityMiddleware")
	}
}

// ══════════════════════════════════════════
//  TESTS — RequestIDMiddleware
// ══════════════════════════════════════════

func TestRequestID_AddsHeader(t *testing.T) {
	handler := RequestIDMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)
	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("X-Request-ID devrait être présent dans la réponse")
	}
	if len(id) != 16 {
		t.Errorf("X-Request-ID devrait faire 16 caractères, obtenu %d", len(id))
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	handler := RequestIDMiddleware(okHandler())
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, r)
		id := w.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("ID dupliqué détecté : %s", id)
		}
		ids[id] = true
	}
}

func TestRequestID_AvailableInContext(t *testing.T) {
	var capturedID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := RequestIDMiddleware(inner)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if capturedID == "" || capturedID == "-" {
		t.Error("RequestID devrait être accessible depuis le context")
	}
	if len(capturedID) != 16 {
		t.Errorf("RequestID dans le context devrait faire 16 chars, obtenu %d", len(capturedID))
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	id := GetRequestID(r.Context())
	if id != "-" {
		t.Errorf("context vide devrait retourner '-', obtenu %s", id)
	}
}

// ══════════════════════════════════════════
//  TESTS — CacheMiddleware
// ══════════════════════════════════════════

func TestCache_ImagesLongCache(t *testing.T) {
	handler := CacheMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/img/favicon.png", nil)
	handler.ServeHTTP(w, r)
	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control images incorrect : %s", cc)
	}
}

func TestCache_CSSLongCache(t *testing.T) {
	handler := CacheMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/css/nav.css", nil)
	handler.ServeHTTP(w, r)
	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=604800" {
		t.Errorf("Cache-Control CSS incorrect : %s", cc)
	}
}

func TestCache_JSLongCache(t *testing.T) {
	handler := CacheMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/js/nav.js", nil)
	handler.ServeHTTP(w, r)
	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=604800" {
		t.Errorf("Cache-Control JS incorrect : %s", cc)
	}
}

func TestCache_HTMLNoCache(t *testing.T) {
	handler := CacheMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/home", nil)
	handler.ServeHTTP(w, r)
	cc := w.Header().Get("Cache-Control")
	if cc != "no-cache, no-store, must-revalidate" {
		t.Errorf("Cache-Control HTML incorrect : %s", cc)
	}
}

func TestCache_HTMLPragmaNoCache(t *testing.T) {
	handler := CacheMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/about", nil)
	handler.ServeHTTP(w, r)
	pragma := w.Header().Get("Pragma")
	if pragma != "no-cache" {
		t.Errorf("Pragma devrait être no-cache pour HTML, obtenu %s", pragma)
	}
}

// ══════════════════════════════════════════
//  TESTS — GzipMiddleware
// ══════════════════════════════════════════

func TestGzip_CompressesWhenAccepted(t *testing.T) {
	handler := GzipMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/home", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding devrait être gzip quand Accept-Encoding: gzip")
	}
}

func TestGzip_SkipsWithoutAcceptEncoding(t *testing.T) {
	handler := GzipMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/home", nil)
	handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding ne devrait pas être gzip sans Accept-Encoding")
	}
}

func TestGzip_SkipsImages(t *testing.T) {
	images := []string{"/img/logo.png", "/img/bg.jpg", "/img/icon.ico", "/img/photo.webp"}
	for _, path := range images {
		handler := GzipMiddleware(okHandler())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(w, r)
		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Errorf("les images ne devraient pas être gzippées : %s", path)
		}
	}
}

func TestGzip_VaryHeader(t *testing.T) {
	handler := GzipMiddleware(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/home", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	handler.ServeHTTP(w, r)
	if w.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("Vary: Accept-Encoding devrait être présent avec gzip")
	}
}
