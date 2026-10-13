package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Chaque code d'erreur a sa page, en français ET en anglais
func TestErrorPages_Exist(t *testing.T) {
	chdirToRepoRoot(t)
	codes := append([]int{http.StatusNotFound}, ErrorCodes...)
	for _, code := range codes {
		for _, f := range []string{fmt.Sprintf("error/%d.html", code), fmt.Sprintf("en/error/%d.html", code)} {
			if _, err := os.Stat(templatePath(f)); err != nil {
				t.Errorf("page d'erreur manquante : web/html/%s", f)
			}
		}
	}
}

func TestErrorHandler_StatusLangAndCache(t *testing.T) {
	chdirToRepoRoot(t)
	for _, code := range ErrorCodes {
		for _, tc := range []struct{ path, lang string }{{"/home", "fr"}, {"/en/home", "en"}} {
			// POST : une page d'erreur doit s'afficher quelle que soit la méthode
			r := httptest.NewRequest(http.MethodPost, tc.path, nil)
			w := httptest.NewRecorder()
			ErrorHandler(w, r, code)

			if w.Code != code {
				t.Errorf("%d %s : statut %d", code, tc.path, w.Code)
			}
			if !strings.Contains(w.Body.String(), `<html lang="`+tc.lang+`">`) {
				t.Errorf("%d %s : la page devrait être en %s", code, tc.path, tc.lang)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("%d %s : Cache-Control doit être no-store", code, tc.path)
			}
		}
	}
}

// La 404 vient aussi du dossier error/, dans la bonne langue
func TestNotFound_FromErrorFolder(t *testing.T) {
	chdirToRepoRoot(t)
	for path, lang := range map[string]string{"/nexiste-pas": "fr", "/en/nexiste-pas": "en"} {
		w := httptest.NewRecorder()
		NotFoundHandler(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `<html lang="`+lang+`">`) {
			t.Errorf("%s : attendu la 404 en %s, obtenu %d", path, lang, w.Code)
		}
	}
}

// Une méthode non autorisée sur une page affiche la page 405
func TestRenderPage_MethodNotAllowedPage(t *testing.T) {
	chdirToRepoRoot(t)
	w := httptest.NewRecorder()
	HomeHandler(w, httptest.NewRequest(http.MethodPost, "/home", nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("POST /home : attendu 405 + Allow, obtenu %d %q", w.Code, w.Header().Get("Allow"))
	}
	if !strings.Contains(w.Body.String(), "405") {
		t.Error("la page 405 doit s'afficher")
	}
}
