package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Remplace les fonctions de rendu par des fausses, le temps d'un test
func fakeErrorPages(t *testing.T) *[]int {
	t.Helper()
	var calls []int
	oldErr, oldNF := ErrorPage, NotFoundPage
	ErrorPage = func(w http.ResponseWriter, r *http.Request, status int) {
		calls = append(calls, status)
		w.WriteHeader(status)
	}
	NotFoundPage = func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
	}
	t.Cleanup(func() { ErrorPage, NotFoundPage = oldErr, oldNF })
	return &calls
}

func TestErrors_PanicUsesErrorPage(t *testing.T) {
	_ = captureLogs(t)
	calls := fakeErrorPages(t)
	h := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("boom") }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if len(*calls) != 1 || (*calls)[0] != 500 {
		t.Errorf("la page 500 doit être rendue, appels : %v", *calls)
	}
}

// Les erreurs envoyées avant SecurityMiddleware ont quand même la CSP et no-store
func TestErrors_SecurityHeadersAndNoStore(t *testing.T) {
	_ = fakeErrorPages(t)
	w := httptest.NewRecorder()
	writeError(w, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusTooManyRequests)
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("une page d'erreur doit avoir la CSP")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("une page d'erreur ne doit pas être mise en cache")
	}
}

// Route piège : exactement la même 404 que les autres pages
func TestErrors_HoneypotUsesNormal404(t *testing.T) {
	_ = captureLogs(t)
	calls := fakeErrorPages(t)
	w := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/wp-login.php", "20.1.0.1"))
	if w.Code != http.StatusNotFound || len(*calls) != 1 {
		t.Errorf("honeypot : attendu la 404 du site, obtenu %d (appels %v)", w.Code, *calls)
	}
}

// Une IP blacklistée peut encore charger le CSS de la page 403
func TestErrors_BlacklistedCanLoadAssets(t *testing.T) {
	_ = captureLogs(t)
	_ = fakeErrorPages(t)
	ip := "20.1.0.2"
	HoneypotMiddleware(okHandler()).ServeHTTP(httptest.NewRecorder(), newRequest(http.MethodGet, "/.env", ip))

	page := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(page, newRequest(http.MethodGet, "/home", ip))
	if page.Code != http.StatusForbidden {
		t.Errorf("page : attendu 403, obtenu %d", page.Code)
	}
	css := httptest.NewRecorder()
	HoneypotMiddleware(okHandler()).ServeHTTP(css, newRequest(http.MethodGet, "/css/404.css", ip))
	if css.Code != http.StatusOK {
		t.Errorf("CSS : attendu 200 (sinon la page 403 s'affiche sans style), obtenu %d", css.Code)
	}
}

func TestErrors_RateLimitSkipsAssets(t *testing.T) {
	_ = captureLogs(t)
	ip := "20.1.0.3"
	for i := 0; i < 200; i++ {
		w := httptest.NewRecorder()
		RateLimitMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/fonts/syne.woff2", ip))
		if w.Code != http.StatusOK {
			t.Fatalf("requête %d sur une police : attendu 200, obtenu %d", i+1, w.Code)
		}
	}
}

func TestMaintenance(t *testing.T) {
	calls := fakeErrorPages(t)
	t.Setenv("MAINTENANCE", "true")
	h := MaintenanceMiddleware(okHandler())

	cases := map[string]int{"/home": 503, "/health": 200, "/api/status": 200, "/css/nav.css": 200}
	for path, want := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != want {
			t.Errorf("%s en maintenance : attendu %d, obtenu %d", path, want, w.Code)
		}
		if want == 503 && w.Header().Get("Retry-After") == "" {
			t.Errorf("%s : Retry-After manquant", path)
		}
	}
	if len(*calls) != 1 {
		t.Errorf("une seule page 503 attendue, appels : %v", *calls)
	}
}

func TestMaintenance_OffByDefault(t *testing.T) {
	t.Setenv("MAINTENANCE", "")
	w := httptest.NewRecorder()
	MaintenanceMiddleware(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/home", nil))
	if w.Code != http.StatusOK {
		t.Errorf("sans MAINTENANCE : attendu 200, obtenu %d", w.Code)
	}
}
