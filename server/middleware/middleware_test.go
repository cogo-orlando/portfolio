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

// cfEdge : une IP Cloudflare réelle (plage 173.245.48.0/20)
const cfEdge = "173.245.48.10"

// newRequest simule une requête passée par Cloudflare : le pair est un serveur
// Cloudflare, donc CF-Connecting-IP (l'IP du visiteur) est pris en compte.
func newRequest(method, path, ip string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.RemoteAddr = cfEdge + ":443"
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
//  TESTS — GetIP (IP non falsifiable)
// ══════════════════════════════════════════

// Via Cloudflare : on fait confiance à CF-Connecting-IP
func TestGetIP_ViaCloudflare(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = cfEdge + ":443"
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	if ip := GetIP(r); ip != "1.2.3.4" {
		t.Errorf("attendu 1.2.3.4, obtenu %s", ip)
	}
}

// Derrière Render : le dernier saut de X-Forwarded-For est un serveur Cloudflare
func TestGetIP_ViaCloudflareBehindRender(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:5000" // proxy interne de Render
	r.Header.Set("X-Forwarded-For", "1.2.3.4, "+cfEdge)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	if ip := GetIP(r); ip != "1.2.3.4" {
		t.Errorf("attendu 1.2.3.4, obtenu %s", ip)
	}
}

// L'ATTAQUE : accès direct à onrender.com avec un faux CF-Connecting-IP
func TestGetIP_SpoofedCloudflareHeaderIgnored(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:5000"
	r.Header.Set("X-Forwarded-For", "6.6.6.6")  // ajouté par Render : la vraie IP de l'attaquant
	r.Header.Set("CF-Connecting-IP", "1.2.3.4") // falsifié par l'attaquant
	if ip := GetIP(r); ip != "6.6.6.6" {
		t.Errorf("le faux CF-Connecting-IP doit être ignoré : attendu 6.6.6.6, obtenu %s", ip)
	}
}

// L'ATTAQUE (variante) : faux X-Forwarded-For ajouté devant par l'attaquant
func TestGetIP_SpoofedForwardedForIgnored(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:5000"
	r.Header.Set("X-Forwarded-For", cfEdge+", 6.6.6.6") // l'attaquant écrit une IP Cloudflare, Render ajoute la vraie
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	if ip := GetIP(r); ip != "6.6.6.6" {
		t.Errorf("seul le dernier saut (ajouté par le proxy) compte : attendu 6.6.6.6, obtenu %s", ip)
	}
}

// Un CF-Connecting-IP invalide n'est jamais utilisé, même via Cloudflare
func TestGetIP_InvalidCloudflareHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = cfEdge + ":443"
	r.Header.Set("CF-Connecting-IP", "pas-une-ip<script>")
	if ip := GetIP(r); ip != cfEdge {
		t.Errorf("attendu l'IP du pair %s, obtenu %s", cfEdge, ip)
	}
}

// En local (sans proxy) : RemoteAddr, sans le port
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

func TestIsCloudflare(t *testing.T) {
	cases := map[string]bool{
		"173.245.48.10":     true,
		"104.16.0.1":        true,
		"2606:4700::1111":   true,
		"::ffff:104.16.0.1": true, // IPv4 écrite en IPv6
		"8.8.8.8":           false,
		"6.6.6.6":           false,
		"pas-une-ip":        false,
		"":                  false,
	}
	for ip, want := range cases {
		if got := isCloudflare(ip); got != want {
			t.Errorf("isCloudflare(%q) : attendu %v, obtenu %v", ip, want, got)
		}
	}
}

// ══════════════════════════════════════════
//  TESTS — CloudflareOnlyMiddleware
// ══════════════════════════════════════════

func directRequest(path string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.RemoteAddr = "10.0.0.5:5000"
	r.Header.Set("X-Forwarded-For", "6.6.6.6") // pair public, hors Cloudflare
	return r
}

func TestCloudflareOnly_BlocksDirectAccessWhenRequired(t *testing.T) {
	_ = captureLogs(t)
	t.Setenv("REQUIRE_CLOUDFLARE", "true")
	w := httptest.NewRecorder()
	CloudflareOnlyMiddleware(okHandler()).ServeHTTP(w, directRequest("/home"))
	if w.Code != http.StatusForbidden {
		t.Errorf("accès direct avec REQUIRE_CLOUDFLARE : attendu 403, obtenu %d", w.Code)
	}
}

func TestCloudflareOnly_AllowsCloudflare(t *testing.T) {
	t.Setenv("REQUIRE_CLOUDFLARE", "true")
	w := httptest.NewRecorder()
	CloudflareOnlyMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/home", "1.2.3.4"))
	if w.Code != http.StatusOK {
		t.Errorf("requête via Cloudflare : attendu 200, obtenu %d", w.Code)
	}
}

// Le health check de Render n'arrive pas par Cloudflare : /health reste accessible
func TestCloudflareOnly_HealthAlwaysAllowed(t *testing.T) {
	t.Setenv("REQUIRE_CLOUDFLARE", "true")
	w := httptest.NewRecorder()
	CloudflareOnlyMiddleware(okHandler()).ServeHTTP(w, directRequest("/health"))
	if w.Code != http.StatusOK {
		t.Errorf("/health : attendu 200, obtenu %d", w.Code)
	}
}

// En local (loopback / réseau privé), jamais bloqué
func TestCloudflareOnly_LocalAllowed(t *testing.T) {
	t.Setenv("REQUIRE_CLOUDFLARE", "true")
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	r.RemoteAddr = "127.0.0.1:50000"
	w := httptest.NewRecorder()
	CloudflareOnlyMiddleware(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("accès local : attendu 200, obtenu %d", w.Code)
	}
}

// Sans REQUIRE_CLOUDFLARE : l'accès direct est seulement logué
func TestCloudflareOnly_LogOnlyByDefault(t *testing.T) {
	logs := captureLogs(t)
	t.Setenv("REQUIRE_CLOUDFLARE", "")
	w := httptest.NewRecorder()
	CloudflareOnlyMiddleware(okHandler()).ServeHTTP(w, directRequest("/home"))
	if w.Code != http.StatusOK {
		t.Errorf("sans REQUIRE_CLOUDFLARE : attendu 200, obtenu %d", w.Code)
	}
	if !strings.Contains(logs.String(), "accès direct") {
		t.Error("l'accès direct doit être logué")
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

// CSP stricte : aucun script inline, pas d'eval, pas d'iframe externe
func TestSecurity_StrictCSP(t *testing.T) {
	w := httptest.NewRecorder()
	SecurityMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy manquante")
	}
	script := ""
	for _, d := range strings.Split(csp, ";") {
		if strings.HasPrefix(strings.TrimSpace(d), "script-src") {
			script = d
		}
	}
	if script == "" {
		t.Fatal("directive script-src manquante")
	}
	for _, bad := range []string{"'unsafe-inline'", "'unsafe-eval'", "*", "http:"} {
		if strings.Contains(script, bad) {
			t.Errorf("script-src ne doit pas contenir %s : %s", bad, script)
		}
	}
	for _, must := range []string{"frame-ancestors 'none'", "object-src 'none'", "base-uri 'self'"} {
		if !strings.Contains(csp, must) {
			t.Errorf("la CSP doit contenir %q", must)
		}
	}
}

func TestSecurity_HardeningHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SecurityMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for h, v := range want {
		if got := w.Header().Get(h); got != v {
			t.Errorf("%s : attendu %q, obtenu %q", h, v, got)
		}
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

// Polices → 1 an + immutable
func TestCache_FontsImmutable(t *testing.T) {
	want := "public, max-age=31536000, immutable"
	if got := cacheControlFor(t, "/fonts/syne-latin-800-normal.woff2"); got != want {
		t.Errorf("polices : attendu %q, obtenu %q", want, got)
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
