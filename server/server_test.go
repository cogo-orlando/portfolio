package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ══════════════════════════════════════════
//  TESTS — healthHandler
// ══════════════════════════════════════════

func TestHealthHandler_Returns200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("healthHandler devrait retourner 200, obtenu %d", w.Code)
	}
}

func TestHealthHandler_ContentTypeJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type devrait être application/json, obtenu %s", ct)
	}
}

func TestHealthHandler_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("healthHandler devrait retourner du JSON valide : %v", err)
	}
}

func TestHealthHandler_HasRequiredFields(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}

	required := []string{"status", "uptime", "goroutines", "go_version", "memory_mb", "gc_cycles"}
	for _, field := range required {
		if _, ok := result[field]; !ok {
			t.Errorf("healthHandler : champ manquant %q", field)
		}
	}
}

func TestHealthHandler_StatusIsOk(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status devrait être 'ok', obtenu %v", result["status"])
	}
}

func TestHealthHandler_GoroutinesPositive(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	goroutines, ok := result["goroutines"].(float64)
	if !ok || goroutines <= 0 {
		t.Errorf("goroutines devrait être un nombre positif, obtenu %v", result["goroutines"])
	}
}

func TestHealthHandler_ServiceIsPortfolio(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if result["service"] != "portfolio" {
		t.Errorf("service devrait être 'portfolio', obtenu %v", result["service"])
	}
}

// ══════════════════════════════════════════
//  TESTS — sitemapHandler
// ══════════════════════════════════════════

func TestSitemapHandler_Returns200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("sitemapHandler devrait retourner 200, obtenu %d", w.Code)
	}
}

func TestSitemapHandler_ContentTypeXML(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/xml") {
		t.Errorf("Content-Type devrait être application/xml, obtenu %s", ct)
	}
}

func TestSitemapHandler_ContainsURLSet(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "<urlset") {
		t.Error("sitemap devrait contenir <urlset>")
	}
}

func TestSitemapHandler_ContainsRequiredPages(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	body := w.Body.String()

	pages := []string{"/home", "/about", "/skills", "/project", "/contact", "/cv", "/faq", "/tech"}
	for _, page := range pages {
		if !strings.Contains(body, page) {
			t.Errorf("sitemap devrait contenir la page %s", page)
		}
	}
}

func TestSitemapHandler_ContainsDomain(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	if !strings.Contains(w.Body.String(), "orlandocogo.com") {
		t.Error("sitemap devrait contenir le domaine orlandocogo.com")
	}
}

func TestSitemapHandler_ValidXMLDeclaration(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapHandler(w, r)
	if !strings.HasPrefix(w.Body.String(), "<?xml") {
		t.Error("sitemap devrait commencer par la déclaration XML")
	}
}

// ══════════════════════════════════════════
//  TESTS — visitsHandler
// ══════════════════════════════════════════

func TestVisitsHandler_Returns200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/visits", nil)
	visitsHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("visitsHandler devrait retourner 200, obtenu %d", w.Code)
	}
}

func TestVisitsHandler_ContentTypeJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/visits", nil)
	visitsHandler(w, r)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type devrait être application/json, obtenu %s", ct)
	}
}

func TestVisitsHandler_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/visits", nil)
	visitsHandler(w, r)
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("visitsHandler devrait retourner du JSON valide : %v", err)
	}
}

func TestVisitsHandler_HasVisitsField(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/visits", nil)
	visitsHandler(w, r)
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if _, ok := result["visits"]; !ok {
		t.Error("visitsHandler devrait retourner un champ 'visits'")
	}
}

func TestVisitsHandler_VisitsIsNumber(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/visits", nil)
	visitsHandler(w, r)
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if _, ok := result["visits"].(float64); !ok {
		t.Errorf("visits devrait être un nombre, obtenu %T", result["visits"])
	}
}

// ══════════════════════════════════════════
//  TESTS — maintenanceMode
// ══════════════════════════════════════════

func TestMaintenanceMode_DefaultFalse(t *testing.T) {
	if MaintenanceMode {
		t.Error("MaintenanceMode devrait être false par défaut")
	}
}
