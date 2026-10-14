package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goToRepoRoot se place à la racine du projet (là où est go.mod),
// pour que le serveur trouve web/html pendant le test
func goToRepoRoot(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod introuvable")
		}
		dir = parent
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestI18n_EnPath(t *testing.T) {
	cases := map[string]string{"/": "/en", "/home": "/en/home", "/projects/zoo": "/en/projects/zoo"}
	for in, want := range cases {
		if got := enPath(in); got != want {
			t.Errorf("enPath(%q) : attendu %q, obtenu %q", in, want, got)
		}
	}
}

func TestI18n_FrPath(t *testing.T) {
	cases := map[string]string{"/en": "/", "/en/": "/", "/en/home": "/home", "/en/projects/zoo": "/projects/zoo"}
	for in, want := range cases {
		got, ok := frPath(in)
		if !ok || got != want {
			t.Errorf("frPath(%q) : attendu %q, obtenu %q (%v)", in, want, got, ok)
		}
	}
	for _, notEN := range []string{"/home", "/english", "/enx/home"} {
		if _, ok := frPath(notEN); ok {
			t.Errorf("%q n'est pas une page anglaise", notEN)
		}
	}
}

// Page anglaise pas encore traduite → redirection vers la française
func TestI18n_FallbackToFrench(t *testing.T) {
	if got, ok := fallbackTarget("/en/home"); !ok || got != "/home" {
		t.Errorf("/en/home : attendu /home, obtenu %q (%v)", got, ok)
	}
	if got, ok := fallbackTarget("/en/"); !ok || got != "/home" {
		t.Errorf("/en/ sans accueil anglais : attendu /home, obtenu %q (%v)", got, ok)
	}
	translated["/home"] = true
	defer delete(translated, "/home")
	if got, _ := fallbackTarget("/en"); got != "/en/home" {
		t.Errorf("/en avec accueil anglais : attendu /en/home, obtenu %q", got)
	}
}

// Une page qui n'existe dans aucune langue reste une 404 (pas de redirection ouverte)
func TestI18n_NoFallbackForUnknownPages(t *testing.T) {
	for _, p := range []string{"/en/nexiste-pas", "/en//evil.com", "/en/../etc/passwd"} {
		if got, ok := fallbackTarget(p); ok {
			t.Errorf("%q ne doit pas être redirigé (obtenu %q)", p, got)
		}
	}
}

func TestI18n_AlternatesHeader(t *testing.T) {
	h := alternatesHeader("/home")
	for _, want := range []string{
		`<https://orlandocogo.com/home>; rel="alternate"; hreflang="fr"`,
		`<https://orlandocogo.com/en/home>; rel="alternate"; hreflang="en"`,
		`hreflang="x-default"`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("en-tête Link : %q manquant dans %q", want, h)
		}
	}
}

// La racine redirige vers la home, et /en vers la home anglaise
func TestRootRedirects(t *testing.T) {
	goToRepoRoot(t)
	mux := newMux()
	cases := map[string]string{"/": "/home", "/en": "/en/home", "/en/": "/en/home"}
	for from, want := range cases {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, from, nil))
		if w.Code != http.StatusMovedPermanently && w.Code != http.StatusFound {
			t.Errorf("%s : attendu une redirection, obtenu %d", from, w.Code)
		}
		if got := w.Header().Get("Location"); got != want {
			t.Errorf("%s : attendu %s, obtenu %s", from, want, got)
		}
	}
}
