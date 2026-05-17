package handler

import (
	"html/template"
	"net/http"
	"sync"
)

// ══════════════════════════════════════════
//  TEMPLATE CACHE — compile une seule fois
// ══════════════════════════════════════════

var (
	templateCache = make(map[string]*template.Template)
	templateMu    sync.RWMutex
)

func renderTemplate(w http.ResponseWriter, r *http.Request, file string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	templateMu.RLock()
	tmpl, cached := templateCache[file]
	templateMu.RUnlock()

	if !cached {
		var err error
		tmpl, err = template.ParseFiles("web/html/" + file)
		if err != nil {
			http.Error(w, "Page introuvable", http.StatusInternalServerError)
			return
		}
		templateMu.Lock()
		templateCache[file] = tmpl
		templateMu.Unlock()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Erreur de rendu", http.StatusInternalServerError)
	}
}

// ══════════════════════════════════════════
//  PAGES
// ══════════════════════════════════════════

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "index.html")
}
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "home.html")
}
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "about.html")
}
func SkillsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "skills.html")
}
func ProjectHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "project.html")
}
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "contact.html")
}
func CvHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "cv.html")
}
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "status.html")
}
func FaqHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "faq.html")
}
func TechHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "tech.html")
}
func MaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "maintenance.html")
}
func ZooHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/zoo.html")
}
func NetflixHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/netflix.html")
}
func GroupieHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/groupie.html")
}
func Power4Handler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/power4.html")
}
func CiscoHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/cisco.html")
}
func ArtemisHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/artemis.html")
}
func AnnuaireHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/annuaire.html")
}

func SecurityDashboardHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/security-dashboard.html")
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, r, "404.html")
}
