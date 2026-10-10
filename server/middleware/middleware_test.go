package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
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
		_, _ = w.Write([]byte("ok"))
	})
}

func newRequest(method, path, ip string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("CF-Connecting-IP", ip)
	return r
}

// fullWindow renvoie n horodatages "maintenant" (pour simuler n requêtes récentes)
func fullWindow(n int, at time.Time) []time.Time {
	times := make([]time.Time, n)
	for i := range times {
		times[i] = at
	}
	return times
}

func resetRateLimiter() {
	globalRateLimiter = &globalLimiter{records: make(map[string][]time.Time)}
}

func resetBlacklist() {
	ipBlacklist = &blacklist{records: make(map[string]time.Time)}
}

// cacheControlFor exécute CacheMiddleware sur une URL et renvoie le Cache-Control obtenu
func cacheControlFor(t *testing.T, target string) string {
	t.Helper()
	w := httptest.NewRecorder()
	CacheMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w.Header().Get("Cache-Control")
}

// captureLogs redirige slog vers un buffer le temps d'un test
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// ══════════════════════════════════════════
//  TESTS — GetIP
// ══════════════════════════════════════════

func TestGetIP_CloudflareHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	if ip := GetIP(r); ip != "1.2.3.4" {
		t.Errorf("attendu 1.2.3.4, obtenu %s", ip)
	}
}

func TestGetIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "5.6.7.8, 9.9.9.9")
	if ip := GetIP(r); ip != "5.6.7.8" {
		t.Errorf("attendu 5.6.7.8 (premier de la liste), obtenu %s", ip)
	}
}

// RemoteAddr contient "ip:port" : seul l'IP doit être gardée.
// Sinon chaque nouvelle connexion (nouveau port) contournerait le rate limit et le honeypot.
func TestGetIP_RemoteAddrStripsPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:1234"
	if ip := GetIP(r); ip != "192.168.1.1" {
		t.Errorf("attendu 192.168.1.1 (sans le port), obtenu %s", ip)
	}
}

func TestGetIP_RemoteAddrIPv6(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[::1]:8080"
	if ip := GetIP(r); ip != "::1" {
		t.Errorf("attendu ::1, obtenu %s", ip)
	}
}

func TestGetIP_SameIPDifferentPorts(t *testing.T) {
	a := httptest.NewRequest(http.MethodGet, "/", nil)
	a.RemoteAddr = "203.0.113.5:40000"
	b := httptest.NewRequest(http.MethodGet, "/", nil)
	b.RemoteAddr = "203.0.113.5:40001"
	if GetIP(a) != GetIP(b) {
		t.Errorf("deux ports différents doivent donner la même IP : %s vs %s", GetIP(a), GetIP(b))
	}
}

func TestGetIP_CloudflarePriority(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.1.1.1")
	r.Header.Set("X-Forwarded-For", "2.2.2.2")
	r.RemoteAddr = "3.3.3.3:80"
	if ip := GetIP(r); ip != "1.1.1.1" {
		t.Errorf("CF-Connecting-IP devrait avoir la priorité, obtenu %s", ip)
	}
}

// ══════════════════════════════════════════
//  TESTS — RateLimitMiddleware
// ══════════════════════════════════════════

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	resetRateLimiter()
	handler := RateLimitMiddleware(okHandler())
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(http.MethodGet, "/", "10.0.0.1"))
		if w.Code != http.StatusOK {
			t.Errorf("requête %d : attendu 200, obtenu %d", i+1, w.Code)
		}
	}
}

func TestRateLimit_BlocksAfterLimit(t *testing.T) {
	resetRateLimiter()
	globalRateLimiter.records["10.0.0.2"] = fullWindow(120, time.Now())
	w := httptest.NewRecorder()
	RateLimitMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/", "10.0.0.2"))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("attendu 429, obtenu %d", w.Code)
	}
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	resetRateLimiter()
	globalRateLimiter.records["10.0.0.6"] = fullWindow(120, time.Now())
	w := httptest.NewRecorder()
	RateLimitMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/", "10.0.0.6"))
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After devrait valoir 60 sur un 429, obtenu %q", got)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	resetRateLimiter()
	globalRateLimiter.records["10.0.1.1"] = fullWindow(120, time.Now())
	w := httptest.NewRecorder()
	RateLimitMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/", "10.0.1.2"))
	if w.Code != http.StatusOK {
		t.Errorf("IP B ne devrait pas être bloquée, obtenu %d", w.Code)
	}
}

func TestRateLimit_ExpiredRequestsNotCounted(t *testing.T) {
	resetRateLimiter()
	globalRateLimiter.records["10.0.0.7"] = fullWindow(120, time.Now().Add(-2*time.Minute))
	w := httptest.NewRecorder()
	RateLimitMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/", "10.0.0.7"))
	if w.Code != http.StatusOK {
		t.Errorf("requêtes expirées ne devraient pas compter, obtenu %d", w.Code)
	}
}

func TestRateLimit_AllowMethod(t *testing.T) {
	resetRateLimiter()
	globalRateLimiter.records["10.0.0.8"] = fullWindow(119, time.Now())
	if !globalRateLimiter.allow("10.0.0.8") {
		t.Error("la 120e requête devrait encore être autorisée")
	}
	if globalRateLimiter.allow("10.0.0.8") {
		t.Error("la 121e requête devrait être refusée")
	}
}

// ══════════════════════════════════════════
//  TESTS — HoneypotMiddleware
// ══════════════════════════════════════════

func TestHoneypot_NormalRoute(t *testing.T) {
	resetBlacklist()
	w := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/home", "20.0.0.1"))
	if w.Code != http.StatusOK {
		t.Errorf("/home devrait passer, obtenu %d", w.Code)
	}
}

func TestHoneypot_BlacklistsOnTrap(t *testing.T) {
	resetBlacklist()
	ip := "20.0.0.2"
	w := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/wp-admin", ip))
	if w.Code != http.StatusNotFound {
		t.Errorf("honeypot devrait retourner 404 (discret), obtenu %d", w.Code)
	}
	if !ipBlacklist.has(ip) {
		t.Error("l'IP devrait être blacklistée après avoir touché le honeypot")
	}
}

func TestHoneypot_BlocksBlacklistedIP(t *testing.T) {
	resetBlacklist()
	ipBlacklist.add("20.0.0.3")
	w := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/home", "20.0.0.3"))
	if w.Code != http.StatusForbidden {
		t.Errorf("IP blacklistée devrait obtenir 403, obtenu %d", w.Code)
	}
}

func TestHoneypot_AllTraps(t *testing.T) {
	for path := range honeypotRoutes {
		resetBlacklist()
		w := httptest.NewRecorder()
		HoneypotMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, path, "30.0.0.1"))
		if w.Code == http.StatusOK {
			t.Errorf("route honeypot %s ne devrait pas retourner 200", path)
		}
		if !ipBlacklist.has("30.0.0.1") {
			t.Errorf("route honeypot %s devrait blacklister l'IP", path)
		}
	}
}

func TestHoneypot_TrapCount(t *testing.T) {
	if len(honeypotRoutes) != 15 {
		t.Errorf("le site annonce 15 routes piège, il y en a %d (mettre à jour _data.html)", len(honeypotRoutes))
	}
}

func TestHoneypot_BlacklistExpiry(t *testing.T) {
	resetBlacklist()
	ipBlacklist.mu.Lock()
	ipBlacklist.records["20.0.0.9"] = time.Now().Add(-1 * time.Hour)
	ipBlacklist.mu.Unlock()
	if ipBlacklist.has("20.0.0.9") {
		t.Error("IP avec expiration passée ne devrait pas être blacklistée")
	}
}

func TestHoneypot_BlacklistAdd(t *testing.T) {
	resetBlacklist()
	ipBlacklist.add("20.0.0.10")
	if !ipBlacklist.has("20.0.0.10") {
		t.Error("IP ajoutée devrait être dans la blacklist")
	}
}

func TestHoneypot_BlacklistFresh(t *testing.T) {
	resetBlacklist()
	if ipBlacklist.has("20.0.0.11") {
		t.Error("IP non ajoutée ne devrait pas être dans la blacklist")
	}
}

func TestHoneypot_TrapThenBlocked(t *testing.T) {
	resetBlacklist()
	handler := HoneypotMiddleware(okHandler())
	ip := "20.0.0.12"
	handler.ServeHTTP(httptest.NewRecorder(), newRequest(http.MethodGet, "/wp-admin", ip))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, newRequest(http.MethodGet, "/home", ip))
	if w.Code != http.StatusForbidden {
		t.Errorf("après honeypot, /home devrait être 403, obtenu %d", w.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — RecoveryMiddleware
// ══════════════════════════════════════════

func TestRecovery_NormalHandler(t *testing.T) {
	w := httptest.NewRecorder()
	RecoveryMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("attendu 200, obtenu %d", w.Code)
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	_ = captureLogs(t) // évite d'inonder la sortie des tests avec la stack trace
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("test panic") })
	w := httptest.NewRecorder()
	RecoveryMiddleware(panicHandler).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRecovery_CatchesNilPointer(t *testing.T) {
	_ = captureLogs(t)
	nilHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s *string
		_ = *s
	})
	w := httptest.NewRecorder()
	RecoveryMiddleware(nilHandler).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("nil pointer devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRecovery_BodyDoesNotLeakPanic(t *testing.T) {
	_ = captureLogs(t)
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("secret interne") })
	w := httptest.NewRecorder()
	RecoveryMiddleware(panicHandler).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, "500") {
		t.Error("le body devrait contenir '500' après un panic")
	}
	if strings.Contains(body, "secret interne") || strings.Contains(body, "goroutine") {
		t.Error("le message du panic et la stack trace ne doivent jamais être envoyés au client")
	}
}

func TestRecovery_LogsStack(t *testing.T) {
	logs := captureLogs(t)
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("boom") })
	RecoveryMiddleware(panicHandler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(logs.String(), "boom") || !strings.Contains(logs.String(), "stack") {
		t.Error("le panic et sa stack trace doivent être logués côté serveur")
	}
}

// ══════════════════════════════════════════
//  TESTS — SecurityMiddleware
// ══════════════════════════════════════════

func TestSecurity_ServerHeaderEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	SecurityMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := w.Header().Get("Server"); got != "" {
		t.Errorf("le header Server devrait être vide, obtenu %q", got)
	}
}

func TestSecurity_XPoweredByRemoved(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Powered-By", "Go")
	SecurityMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Header().Get("X-Powered-By") != "" {
		t.Error("X-Powered-By devrait être supprimé par SecurityMiddleware")
	}
}

// ══════════════════════════════════════════
//  TESTS — RequestIDMiddleware
// ══════════════════════════════════════════

func TestRequestID_AddsHeader(t *testing.T) {
	w := httptest.NewRecorder()
	RequestIDMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	id := w.Header().Get("X-Request-ID")
	if len(id) != 16 {
		t.Errorf("X-Request-ID devrait faire 16 caractères, obtenu %q", id)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	handler := RequestIDMiddleware(okHandler())
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		id := w.Header().Get("X-Request-ID")
		if ids[id] {
			t.Fatalf("ID dupliqué détecté : %s", id)
		}
		ids[id] = true
	}
}

func TestRequestID_AvailableInContext(t *testing.T) {
	var captured string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetRequestID(r.Context())
	})
	RequestIDMiddleware(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if len(captured) != 16 {
		t.Errorf("RequestID dans le context devrait faire 16 caractères, obtenu %q", captured)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	if id := GetRequestID(httptest.NewRequest(http.MethodGet, "/", nil).Context()); id != "-" {
		t.Errorf("context vide devrait retourner '-', obtenu %s", id)
	}
}

// ══════════════════════════════════════════
//  TESTS — Chain (ordre des middlewares)
// ══════════════════════════════════════════

// Le log de requête doit contenir le vrai request_id (RequestID doit passer AVANT Logger)
func TestChain_LoggerSeesRequestID(t *testing.T) {
	resetRateLimiter()
	resetBlacklist()
	logs := captureLogs(t)

	w := httptest.NewRecorder()
	Chain(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/home", "40.0.0.1"))

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("la Chain doit poser X-Request-ID")
	}
	if !strings.Contains(logs.String(), `"request_id":"`+id+`"`) {
		t.Errorf("le log de requête devrait contenir request_id=%s, logs : %s", id, logs.String())
	}
}

// Un panic dans une route doit être rattrapé ET logué avec un statut 500
func TestChain_PanicReturns500(t *testing.T) {
	resetRateLimiter()
	resetBlacklist()
	_ = captureLogs(t)
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("route cassée") })

	w := httptest.NewRecorder()
	Chain(panicHandler).ServeHTTP(w, newRequest(http.MethodGet, "/home", "40.0.0.2"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("un panic dans une route devrait donner 500, obtenu %d", w.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — CacheMiddleware
// ══════════════════════════════════════════

func TestCache_ImagesImmutable(t *testing.T) {
	want := "public, max-age=31536000, immutable"
	if got := cacheControlFor(t, "/img/favicon.png"); got != want {
		t.Errorf("images : attendu %q, obtenu %q", want, got)
	}
}

// CSS versionné (?v=hash) → 1 an + immutable
func TestCache_CSSVersionedImmutable(t *testing.T) {
	want := "public, max-age=31536000, immutable"
	if got := cacheControlFor(t, "/css/nav.css?v=3f9a1c2b7d"); got != want {
		t.Errorf("CSS versionné : attendu %q, obtenu %q", want, got)
	}
}

// JS versionné (?v=hash) → 1 an + immutable
func TestCache_JSVersionedImmutable(t *testing.T) {
	want := "public, max-age=31536000, immutable"
	if got := cacheControlFor(t, "/js/core.js?v=a1b2c3d4e5"); got != want {
		t.Errorf("JS versionné : attendu %q, obtenu %q", want, got)
	}
}

// CSS/JS sans version → cache court (1h)
func TestCache_UnversionedAssetShortCache(t *testing.T) {
	want := "public, max-age=3600"
	for _, target := range []string{"/css/nav.css", "/js/nav.js", "/css/nav.css?v="} {
		if got := cacheControlFor(t, target); got != want {
			t.Errorf("%s sans version : attendu %q, obtenu %q", target, want, got)
		}
	}
}

// HTML → no-cache (revalidation avec ETag), surtout pas no-store
func TestCache_HTMLRevalidates(t *testing.T) {
	for _, target := range []string{"/", "/home", "/projects/netflix"} {
		got := cacheControlFor(t, target)
		if got != "no-cache" {
			t.Errorf("%s : attendu %q, obtenu %q", target, "no-cache", got)
		}
		if strings.Contains(got, "no-store") {
			t.Errorf("%s ne doit pas avoir no-store : ça rendrait l'ETag inutile", target)
		}
	}
}

// ══════════════════════════════════════════
//  TESTS — GzipMiddleware
// ══════════════════════════════════════════

func TestGzip_CompressesWhenAccepted(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	GzipMiddleware(okHandler()).ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding devrait être gzip quand Accept-Encoding: gzip")
	}
}

// Le contenu compressé doit se décompresser en la réponse d'origine
func TestGzip_BodyDecompresses(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	GzipMiddleware(okHandler()).ServeHTTP(w, r)

	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("le body n'est pas du gzip valide : %v", err)
	}
	defer gz.Close()
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("décompression impossible : %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("après décompression : attendu %q, obtenu %q", "ok", body)
	}
}

func TestGzip_SkipsWithoutAcceptEncoding(t *testing.T) {
	w := httptest.NewRecorder()
	GzipMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/home", nil))
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding ne devrait pas être gzip sans Accept-Encoding")
	}
}

func TestGzip_SkipsImages(t *testing.T) {
	for _, path := range []string{"/img/logo.png", "/img/bg.jpg", "/img/icon.ico", "/img/photo.webp"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("Accept-Encoding", "gzip")
		GzipMiddleware(okHandler()).ServeHTTP(w, r)
		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Errorf("les images ne devraient pas être gzippées : %s", path)
		}
	}
}

func TestGzip_VaryHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	GzipMiddleware(okHandler()).ServeHTTP(w, r)
	if w.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("Vary: Accept-Encoding devrait être présent avec gzip")
	}
}

// ══════════════════════════════════════════
//  TESTS — Circuit Breaker DB
// ══════════════════════════════════════════

func TestCircuitBreaker_InitiallyClosed(t *testing.T) {
	cb := &dbCircuitBreaker{maxFailures: 5, resetTimeout: 30 * time.Second}
	if !cb.Allow() {
		t.Error("circuit breaker devrait être fermé (Allow=true) au démarrage")
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	_ = captureLogs(t)
	cb := &dbCircuitBreaker{maxFailures: 3, resetTimeout: 30 * time.Second}
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if !cb.IsOpen() {
		t.Error("circuit devrait être ouvert après 3 échecs")
	}
}

func TestCircuitBreaker_BlocksWhenOpen(t *testing.T) {
	_ = captureLogs(t)
	cb := &dbCircuitBreaker{maxFailures: 3, resetTimeout: 30 * time.Second}
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.Allow() {
		t.Error("circuit ouvert devrait bloquer les appels (Allow=false)")
	}
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	_ = captureLogs(t)
	cb := &dbCircuitBreaker{maxFailures: 3, resetTimeout: 30 * time.Second}
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	cb.RecordSuccess()
	if cb.IsOpen() {
		t.Error("circuit devrait être fermé après un succès")
	}
}

func TestCircuitBreaker_AllowsAfterTimeout(t *testing.T) {
	_ = captureLogs(t)
	cb := &dbCircuitBreaker{maxFailures: 3, resetTimeout: 50 * time.Millisecond}
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Error("circuit devrait permettre un appel test (half-open) après le timeout")
	}
}

func TestCircuitBreaker_NotOpenBeforeMaxFailures(t *testing.T) {
	cb := &dbCircuitBreaker{maxFailures: 5, resetTimeout: 30 * time.Second}
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsOpen() {
		t.Error("circuit ne devrait pas être ouvert avec seulement 2 échecs sur 5")
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := &dbCircuitBreaker{maxFailures: 5, resetTimeout: 30 * time.Second}
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsOpen() {
		t.Error("après RecordSuccess, 2 nouveaux échecs ne devraient pas ouvrir le circuit")
	}
}

// ══════════════════════════════════════════
//  TESTS — TimeoutMiddleware
// ══════════════════════════════════════════

func TestTimeout_PerPath(t *testing.T) {
	cases := map[string]time.Duration{
		"/api/visits":       3 * time.Second,
		"/health":           3 * time.Second,
		"/projects/netflix": 8 * time.Second,
		"/home":             5 * time.Second,
		"/about":            5 * time.Second,
	}
	for path, want := range cases {
		if got := timeoutForPath(path); got != want {
			t.Errorf("%s : timeout attendu %v, obtenu %v", path, want, got)
		}
	}
}

func TestTimeoutMiddleware_PassesNormalRequest(t *testing.T) {
	w := httptest.NewRecorder()
	TimeoutMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/home", nil))
	if w.Code != http.StatusOK {
		t.Errorf("requête normale devrait passer, obtenu %d", w.Code)
	}
}

func TestTimeoutMiddleware_ContextHasDeadline(t *testing.T) {
	var hasDeadline bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasDeadline = r.Context().Deadline()
	})
	TimeoutMiddleware(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/home", nil))
	if !hasDeadline {
		t.Error("le context devrait avoir une deadline après TimeoutMiddleware")
	}
}
