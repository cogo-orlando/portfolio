package handler

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
)

// ══════════════════════════════════════════
//  ENVIRONNEMENT
// ══════════════════════════════════════════

// isDev retourne true si on est en développement
// En prod (Render) la variable GO_ENV n'est pas définie → prod
func isDev() bool {
	return os.Getenv("GO_ENV") == "development"
}

// ══════════════════════════════════════════
//  TEMPLATE CACHE
//  - Dev  : recharge à chaque requête (pas de cache)
//  - Prod : compile une seule fois et met en cache
// ══════════════════════════════════════════

var (
	templateCache = make(map[string]*template.Template)
	templateMu    sync.RWMutex
)

func getTemplate(file string) (*template.Template, error) {
	// Dev — toujours recharger depuis le disque
	if isDev() {
		return template.ParseFiles("web/html/" + file)
	}

	// Prod — utiliser le cache
	templateMu.RLock()
	tmpl, cached := templateCache[file]
	templateMu.RUnlock()

	if cached {
		return tmpl, nil
	}

	var err error
	tmpl, err = template.ParseFiles("web/html/" + file)
	if err != nil {
		return nil, err
	}

	templateMu.Lock()
	templateCache[file] = tmpl
	templateMu.Unlock()

	return tmpl, nil
}

// ══════════════════════════════════════════
//  RENDER TEMPLATE + ETAG
// ══════════════════════════════════════════

func renderTemplate(w http.ResponseWriter, r *http.Request, file string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := getTemplate(file)
	if err != nil {
		http.Error(w, "Page introuvable", http.StatusInternalServerError)
		return
	}

	// ── ETag — hash du nom de fichier + date de modif ──
	etag := generateETag(file)
	w.Header().Set("ETag", etag)

	// Si le client a déjà cette version → 304 Not Modified
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Erreur de rendu", http.StatusInternalServerError)
	}
}

// generateETag génère un ETag stable basé sur le fichier et sa date de modification
func generateETag(file string) string {
	info, err := os.Stat("web/html/" + file)
	if err != nil {
		// Fallback sur le nom seul
		return fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(file)))
	}
	raw := fmt.Sprintf("%s:%d", file, info.ModTime().UnixNano())
	return fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(raw)))
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

func ForumHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/forum.html")
}

func SnakeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/snake.html")
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, r, "404.html")
}
