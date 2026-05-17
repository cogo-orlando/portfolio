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
// ══════════════════════════════════════════

func setupTestTemplates(t *testing.T) func() {
	t.Helper()

	dir := filepath.Join("web", "html")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("impossible de créer %s : %v", dir, err)
	}

	pages := []string{
		"index.html", "home.html", "about.html", "skills.html",
		"project.html", "contact.html", "cv.html", "status.html",
		"faq.html", "tech.html", "maintenance.html", "404.html",
		"projects/zoo.html", "projects/netflix.html", "projects/groupie.html",
		"projects/power4.html", "projects/cisco.html", "projects/artemis.html",
		"projects/annuaire.html", "projects/security-dashboard.html",
	}

	created := []string{}
	if err := os.MkdirAll(filepath.Join("web", "html", "projects"), 0755); err != nil {
		t.Fatalf("impossible de créer web/html/projects : %v", err)
	}
	for _, page := range pages {
		path := filepath.Join(dir, page)
		content := "<!DOCTYPE html><html><body>test:" + page + "</body></html>"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("impossible de créer %s : %v", path, err)
		}
		created = append(created, path)
	}

	templateMu.Lock()
	templateCache = make(map[string]*template.Template)
	templateMu.Unlock()

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
//  HELPER
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

func testHandlerMethod(t *testing.T, handler http.HandlerFunc, method, path string, expectedCode int) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, nil)
	handler(w, r)
	if w.Code != expectedCode {
		t.Errorf("%s %s : attendu %d, obtenu %d", method, path, expectedCode, w.Code)
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

func TestRenderTemplate_RejectsDelete(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/home", nil)
	renderTemplate(w, r, "home.html")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE devrait retourner 405, obtenu %d", w.Code)
	}
}

func TestRenderTemplate_RejectsPatch(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/about", nil)
	renderTemplate(w, r, "about.html")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH devrait retourner 405, obtenu %d", w.Code)
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

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w1, r1, "home.html")

	templateMu.RLock()
	_, cached := templateCache["home.html"]
	templateMu.RUnlock()
	if !cached {
		t.Error("le template devrait être en cache après le premier appel")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/home", nil)
	renderTemplate(w2, r2, "home.html")
	if w2.Code != http.StatusOK {
		t.Errorf("deuxième appel depuis le cache devrait retourner 200, obtenu %d", w2.Code)
	}
}

func TestRenderTemplate_BodyContainsContent(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/about", nil)
	renderTemplate(w, r, "about.html")
	if !strings.Contains(w.Body.String(), "test:about.html") {
		t.Error("le body devrait contenir le contenu du template")
	}
}

// ══════════════════════════════════════════
//  TESTS — tous les handlers GET 200
// ══════════════════════════════════════════

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
		{"Tech", TechHandler, "/tech"},
		{"Maintenance", MaintenanceHandler, "/maintenance"},
		{"Zoo", ZooHandler, "/projects/zoo"},
		{"Netflix", NetflixHandler, "/projects/netflix"},
		{"Groupie", GroupieHandler, "/projects/groupie"},
		{"Power4", Power4Handler, "/projects/power4"},
		{"Cisco", CiscoHandler, "/projects/cisco"},
		{"Artemis", ArtemisHandler, "/projects/artemis"},
		{"Annuaire", AnnuaireHandler, "/projects/annuaire"},
		{"SecurityDashboard", SecurityDashboardHandler, "/projects/security-dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler(t, tt.handler, tt.path)
		})
	}
}

// ══════════════════════════════════════════
//  TESTS — HEAD sur tous les handlers
// ══════════════════════════════════════════

func TestHandlers_HeadReturn200(t *testing.T) {
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
		{"Tech", TechHandler, "/tech"},
		{"SecurityDashboard", SecurityDashboardHandler, "/projects/security-dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandlerMethod(t, tt.handler, http.MethodHead, tt.path, http.StatusOK)
		})
	}
}

// ══════════════════════════════════════════
//  TESTS — POST refusé sur tous les handlers
// ══════════════════════════════════════════

func TestHandlers_PostReturns405(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"Home", HomeHandler, "/home"},
		{"About", AboutHandler, "/about"},
		{"Skills", SkillsHandler, "/skills"},
		{"Contact", ContactHandler, "/contact"},
		{"CV", CvHandler, "/cv"},
		{"FAQ", FaqHandler, "/faq"},
		{"Tech", TechHandler, "/tech"},
		{"SecurityDashboard", SecurityDashboardHandler, "/projects/security-dashboard"},
		{"Cisco", CiscoHandler, "/projects/cisco"},
		{"Annuaire", AnnuaireHandler, "/projects/annuaire"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandlerMethod(t, tt.handler, http.MethodPost, tt.path, http.StatusMethodNotAllowed)
		})
	}
}

// ══════════════════════════════════════════
//  TESTS — NotFoundHandler
// ══════════════════════════════════════════

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

func TestNotFoundHandler_BodyContains404(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/page-inexistante", nil)
	NotFoundHandler(w, r)
	if !strings.Contains(w.Body.String(), "test:404.html") {
		t.Error("NotFoundHandler devrait servir le template 404.html")
	}
}

func TestNotFoundHandler_PostReturns405(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/inexistant", nil)
	NotFoundHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("NotFoundHandler POST devrait retourner 405, obtenu %d", w.Code)
	}
}

// ══════════════════════════════════════════
//  TESTS — Content-Type sur handlers projets
// ══════════════════════════════════════════

func TestProjectHandlers_ContentType(t *testing.T) {
	cleanup := setupTestTemplates(t)
	defer cleanup()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"Cisco", CiscoHandler, "/projects/cisco"},
		{"Artemis", ArtemisHandler, "/projects/artemis"},
		{"Annuaire", AnnuaireHandler, "/projects/annuaire"},
		{"SecurityDashboard", SecurityDashboardHandler, "/projects/security-dashboard"},
		{"Zoo", ZooHandler, "/projects/zoo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			tt.handler(w, r)
			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Errorf("%s : Content-Type devrait être text/html, obtenu %s", tt.path, ct)
			}
		})
	}
}
