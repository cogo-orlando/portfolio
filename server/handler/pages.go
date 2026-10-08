package handler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Dossier des templates HTML (relatif au dossier de lancement du serveur)
const htmlDir = "web/html"

// ══════════════════════════════════════════
//  ENVIRONNEMENT
// ══════════════════════════════════════════

// isDev retourne true si on est en développement.
// En prod (Render) la variable GO_ENV n'est pas définie → prod.
// En local : $env:GO_ENV="development"; go run Main.go
func isDev() bool {
	return os.Getenv("GO_ENV") == "development"
}

// IsDev expose le mode au reste de l'application (logs de démarrage).
func IsDev() bool {
	return isDev()
}

// ══════════════════════════════════════════
//  TEMPLATE CACHE
//  - Dev  : recharge à chaque requête (les modifs HTML sont visibles direct)
//  - Prod : compile une seule fois et met en cache
// ══════════════════════════════════════════

var (
	templateCache = make(map[string]*template.Template)
	templateMu    sync.RWMutex
)

// templatePath convertit "projects/zoo.html" en chemin système valide (Windows et Linux).
func templatePath(file string) string {
	return filepath.Join(htmlDir, filepath.FromSlash(file))
}

// partialFiles liste les morceaux partagés (web/html/partials/*.html).
// Renvoie une liste vide (et pas une erreur) si le dossier n'existe pas encore :
// les pages sans partials continuent de fonctionner normalement.
func partialFiles() []string {
	matches, err := filepath.Glob(filepath.Join(htmlDir, "partials", "*.html"))
	if err != nil {
		slog.Error("lecture des partials", "error", err)
		return nil
	}
	return matches
}

// parseTemplate compile une page + tous les partials.
// La page est parsée EN PREMIER : c'est elle qui est rendue par Execute,
// les partials ne font qu'ajouter des {{define}} utilisables avec {{template "nom" .}}.
func parseTemplate(file string) (*template.Template, error) {
	tmpl, err := template.ParseFiles(templatePath(file))
	if err != nil {
		return nil, err
	}
	if partials := partialFiles(); len(partials) > 0 {
		if tmpl, err = tmpl.ParseFiles(partials...); err != nil {
			return nil, fmt.Errorf("partials: %w", err)
		}
	}
	return tmpl, nil
}

func getTemplate(file string) (*template.Template, error) {
	if isDev() {
		return parseTemplate(file)
	}

	templateMu.RLock()
	tmpl, cached := templateCache[file]
	templateMu.RUnlock()
	if cached {
		return tmpl, nil
	}

	tmpl, err := parseTemplate(file)
	if err != nil {
		return nil, err
	}

	templateMu.Lock()
	templateCache[file] = tmpl
	templateMu.Unlock()

	return tmpl, nil
}

// PreloadTemplates compile tous les templates au démarrage (prod uniquement).
// Un HTML cassé ou introuvable est détecté au lancement
// au lieu de faire planter la page à la première visite.
func PreloadTemplates() (int, error) {
	if isDev() {
		return 0, nil
	}

	count := 0
	for _, pattern := range []string{"*.html", "projects/*.html"} {
		matches, err := filepath.Glob(filepath.Join(htmlDir, pattern))
		if err != nil {
			return count, err
		}
		for _, match := range matches {
			rel, err := filepath.Rel(htmlDir, match)
			if err != nil {
				return count, err
			}
			key := filepath.ToSlash(rel)
			if _, err := getTemplate(key); err != nil {
				return count, fmt.Errorf("template %s: %w", key, err)
			}
			count++
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("aucun template trouvé dans %s — le serveur est-il lancé depuis la racine du projet ?", htmlDir)
	}
	return count, nil
}

// ══════════════════════════════════════════
//  RENDER
// ══════════════════════════════════════════

// renderTemplate rend une page avec un statut 200.
func renderTemplate(w http.ResponseWriter, r *http.Request, file string) {
	renderPage(w, r, file, http.StatusOK)
}

// renderPage rend une page avec le statut HTTP voulu.
// Le HTML est d'abord généré en mémoire : en cas d'erreur de rendu,
// le visiteur reçoit une vraie 500 au lieu d'une page coupée en deux.
func renderPage(w http.ResponseWriter, r *http.Request, file string, status int) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := getTemplate(file)
	if err != nil {
		slog.Error("template introuvable", "file", file, "error", err)
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	// ETag uniquement pour les pages normales (pas pour les erreurs)
	if status == http.StatusOK {
		etag := generateETag(file)
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		slog.Error("erreur de rendu", "file", file, "error", err)
		http.Error(w, "Erreur de rendu", http.StatusInternalServerError)
		return
	}

	// Les headers doivent être posés AVANT WriteHeader, sinon ils sont ignorés
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		// Arrive surtout quand le visiteur ferme la page pendant l'envoi
		slog.Debug("écriture de la réponse interrompue", "file", file, "error", err)
	}
}

// generateETag génère un ETag basé sur la page ET les partials.
// Si tu modifies seulement la nav (partial), l'ETag change quand même :
// le navigateur ne garde pas l'ancienne version en cache.
func generateETag(file string) string {
	raw := file
	for _, path := range append([]string{templatePath(file)}, partialFiles()...) {
		if info, err := os.Stat(path); err == nil {
			raw += fmt.Sprintf(":%d", info.ModTime().UnixNano())
		}
	}
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

// ── Pages projets ──

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

func MotoGPHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/motogp.html")
}

// ── 404 ──

// NotFoundHandler rend la page 404 avec un vrai statut 404.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	renderPage(w, r, "404.html", http.StatusNotFound)
}