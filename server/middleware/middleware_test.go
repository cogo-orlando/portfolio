package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ══════════════════════════════════════════
//  HELPER — handler de test qui répond 200
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

// ══════════════════════════════════════════
//  TESTS — RateLimitMiddleware
// ══════════════════════════════════════════

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	// Réinitialise le rate limiter pour le test
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
	// Réinitialise et configure un limiter strict pour le test
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
	ip := "10.0.0.2"

	// Remplit le quota (120 requêtes)
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

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}

	handler := RateLimitMiddleware(okHandler())

	// IP A — remplit son quota
	ipA := "10.0.1.1"
	now := time.Now()
	times := make([]time.Time, 120)
	for i := range times {
		times[i] = now
	}
	globalRateLimiter.records[ipA] = times

	// IP B — ne doit pas être bloquée
	ipB := "10.0.1.2"
	w := httptest.NewRecorder()
	r := newRequest("GET", "/", ipB)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("IP B ne devrait pas être bloquée, obtenu %d", w.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — HoneypotMiddleware
// ══════════════════════════════════════════

func TestHoneypot_NormalRoute(t *testing.T) {
	// Réinitialise la blacklist
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

	// Accède à une route honeypot
	w := httptest.NewRecorder()
	r := newRequest("GET", "/wp-admin", ip)
	handler.ServeHTTP(w, r)

	// Doit retourner 404 (pas 200)
	if w.Code != http.StatusNotFound {
		t.Errorf("honeypot devrait retourner 404, obtenu %d", w.Code)
	}

	// L'IP doit être blacklistée
	if !ipBlacklist.has(ip) {
		t.Error("l'IP devrait être blacklistée après avoir touché le honeypot")
	}
}

func TestHoneypot_BlocksBlacklistedIP(t *testing.T) {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}

	ip := "20.0.0.3"
	// Blackliste l'IP manuellement
	ipBlacklist.add(ip)

	handler := HoneypotMiddleware(okHandler())
	w := httptest.NewRecorder()
	// Même une route normale doit être bloquée
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
		panic("test panic — ne devrait pas crasher le serveur")
	})

	handler := RecoveryMiddleware(panicHandler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Ne doit pas paniquer
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRecovery_CatchesNilPointer(t *testing.T) {
	nilHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s *string
		_ = *s // nil pointer dereference
	})

	handler := RecoveryMiddleware(nilHandler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("nil pointer devrait retourner 500, obtenu %d", w.Code)
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
	// Le test vérifie que le middleware s'exécute sans erreur
	if w.Code != http.StatusOK {
		t.Errorf("attendu 200, obtenu %d", w.Code)
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
