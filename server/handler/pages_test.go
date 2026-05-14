package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════
//  SETUP — crée des templates HTML de test
//  Les handlers cherchent les fichiers dans
//  "web/html/" — on crée une structure temp
// ══════════════════════════════════════════

func setupTestTemplates(t *testing.T) func() {
	t.Helper()

	// Crée le dossier web/html/ relatif au package handler
	dir := filepath.Join("web", "html")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("impossible de créer %s : %v", dir, err)
	}

	// Liste des templates à créer pour les tests
	pages := []string{
		"index.html", "home.html", "about.html", "skills.html",
		"project.html", "contact.html", "cv.html", "status.html",
		"faq.html", "maintenance.html", "404.html",
		"zoo.html", "netflix.html", "groupie.html",
		"power4.html", "cisco.html", "artemis.html",
		"annuaire.html",
	}

	created := []string{}
	for _, page := range pages {
		path := filepath.Join(dir, page)
		content := "<!DOCTYPE html><html><body>test:" + page + "</body></html>"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("impossible de créer %s : %v", path, err)
		}
		created = append(created, path)
	}

	// Vide le cache entre les tests
	templateMu.Lock()
	templateCache = make(map[string]*template.Template)
	templateMu.Unlock()

	// Retourne la fonction de cleanup
	return func() {
		for _, p := range created {
			os.Remove(p)
		}
		os.RemoveAll("web")
		templateMu.Lock()
		templateCache = make(map[string]*template.Template)
		templateMu.Unlock()
	}
}

// ══════════════════════════════════════════
//  TESTS — renderTemplate
// ══════════════════════════════════════════

func TestRenderTemplate_RejectsPost(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/home", nil)
	renderTemplate(w, r, "home.html")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST devrait retourner 405, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_RejectsPut(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/home", nil)
	renderTemplate(w, r, "home.html")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT devrait retourner 405, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_AcceptsGet(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w, r, "home.html")

	if w.Code != http.StatusOK {
		t.Errorf("GET devrait retourner 200, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_AcceptsHead(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/home", nil)
	renderTemplate(w, r, "home.html")

	if w.Code != http.StatusOK {
		t.Errorf("HEAD devrait retourner 200, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_SetsContentType(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w, r, "home.html")

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type devrait être text/html, obtenu %s", ct)
	}
}

func TestRenderTemplate_MissingFile(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	renderTemplate(w, r, "inexistant.html")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("fichier manquant devrait retourner 500, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_UsesCache(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	// Premier appel — compile le template
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w1, r1, "home.html")

	templateMu.RLock()
	_, cached := templateCache["home.html"]
	templateMu.RUnlock()

	if !cached {
		t.Error("le template devrait être en cache après le premier appel")
	}

	// Deuxième appel — doit utiliser le cache
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w2, r2, "home.html")

	if w2.Code != http.StatusOK {
		t.Errorf("deuxième appel depuis le cache devrait retourner 200, obtenu %d", w2.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — handlers individuels
// ══════════════════════════════════════════

func testHandler(t *testing.T, handler http.HandlerFunc, path string) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	handler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("%s : attendu 200, obtenu %d", path, w.Code)
	}
}

func TestHandlers_AllReturn200(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"Index", IndexHandler, "/"},
		{"Home", HomeHandler, "/home"},
		{"About", AboutHandler, "/about"},
		{"Skills", SkillsHandler, "/skills"},
		{"Project", ProjectHandler, "/project"},
		{"Contact", ContactHandler, "/contact"},
		{"CV", CvHandler, "/cv"},
		{"Status", StatusHandler, "/status"},
		{"FAQ", FaqHandler, "/faq"},
		{"Maintenance", MaintenanceHandler, "/maintenance"},
		{"DemoZoo", DemoZooHandler, "/demo/zoo"},
		{"DemoNetflix", DemoNetflixHandler, "/demo/netflix"},
		{"DemoGroupie", DemoGroupieHandler, "/demo/groupie"},
		{"DemoPower4", DemoPower4Handler, "/demo/power4"},
		{"DemoCisco", DemoCiscoHandler, "/demo/cisco"},
		{"DemoArtemis", DemoArtemisHandler, "/demo/artemis"},
		{"Annuaire", AnnuaireHandler, "/demo/annuaire"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler(t, tt.handler, tt.path)
		})
	}
}

func TestNotFoundHandler_Returns404(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/inexistant", nil)
	NotFoundHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("NotFoundHandler devrait retourner 404, obtenu %d", w.Code)
	}
}
