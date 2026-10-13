package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Dossiers (relatifs au dossier de lancement du serveur)
const (
	htmlDir   = "web/html"
	staticDir = "web"
)

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
//  CACHE-BUSTING : {{asset "/css/nav.css"}} → /css/nav.css?v=3f9a1c2b7d
//  Le hash change dès que le contenu du fichier change :
//  le navigateur télécharge la nouvelle version, sans purge manuelle.
// ══════════════════════════════════════════

var (
	assetCache = make(map[string]string)
	assetMu    sync.RWMutex
)

// assetURL renvoie le chemin public suivi d'un hash court du contenu du fichier.
// En cas de problème (fichier absent, chemin suspect), il renvoie le chemin tel quel :
// la page s'affiche quand même, sans cache-busting pour ce fichier.
func assetURL(publicPath string) string {
	publicPath = strings.TrimSpace(publicPath) // tolère un espace ajouté par un formateur
	clean := path.Clean("/" + publicPath)
	if strings.Contains(publicPath, "..") || clean == "/" {
		slog.Warn("asset : chemin refusé", "path", publicPath)
		return publicPath
	}

	// Prod : le hash est calculé une seule fois (les fichiers ne changent pas pendant l'exécution)
	if !isDev() {
		assetMu.RLock()
		cached, ok := assetCache[clean]
		assetMu.RUnlock()
		if ok {
			return cached
		}
	}

	data, err := os.ReadFile(filepath.Join(staticDir, filepath.FromSlash(clean)))
	if err != nil {
		slog.Warn("asset introuvable", "path", clean, "error", err)
		return clean
	}
	sum := sha256.Sum256(data)
	versioned := clean + "?v=" + hex.EncodeToString(sum[:])[:10]

	if !isDev() {
		assetMu.Lock()
		assetCache[clean] = versioned
		assetMu.Unlock()
	}
	return versioned
}

// templateFuncs : fonctions utilisables dans tous les templates
var templateFuncs = template.FuncMap{
	"asset": assetURL,
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
	// template.New doit porter le nom du fichier : c'est lui qu'Execute rendra.
	// Les fonctions (asset…) doivent être déclarées AVANT le parsing.
	tmpl, err := template.New(filepath.Base(file)).Funcs(templateFuncs).ParseFiles(templatePath(file))
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
		ErrorHandler(w, r, http.StatusMethodNotAllowed)
		return
	}
	writePage(w, r, file, status)
}

// writePage génère la page en mémoire puis l'envoie (sans vérifier la méthode HTTP)
func writePage(w http.ResponseWriter, r *http.Request, file string, status int) {
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

func ForumHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/forum.html")
}

func SnakeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/snake.html")
}

func MotoGPHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "projects/motogp.html")
}

// PageHandler sert un template donné (utilisé pour les pages anglaises : "en/home.html")
func PageHandler(file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, r, file)
	}
}

// ── 404 ──

// NotFoundHandler rend la page 404 avec un vrai statut 404.
// ══════════════════════════════════════════
//  PAGES D'ERREUR (403, 405, 429, 500, 503)
//  Fichiers : web/html/error/<code>.html et web/html/en/error/<code>.html
//  Utilisées par les middlewares (rate limit, honeypot, panic…) via un
//  branchement fait dans server.go.
// ══════════════════════════════════════════

// ErrorCodes : codes qui ont une page dédiée (vérifiés par les tests)
var ErrorCodes = []int{
	http.StatusForbidden,
	http.StatusMethodNotAllowed,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusServiceUnavailable,
}

// errorFile : page d'erreur à afficher (error/404.html, error/429.html…),
// en anglais sous /en/ si elle existe
func errorFile(path string, status int) string {
	file := fmt.Sprintf("error/%d.html", status)
	if isEnglishPath(path) {
		if _, err := os.Stat(templatePath("en/" + file)); err == nil {
			return "en/" + file
		}
	}
	return file
}

// ErrorHandler envoie la page d'erreur du code donné.
// Elle ne doit JAMAIS planter : au moindre souci, réponse texte simple.
func ErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("page d'erreur : panic", "status", status, "error", fmt.Sprint(rec))
			http.Error(w, strconv.Itoa(status)+" — "+http.StatusText(status), status)
		}
	}()
	file := errorFile(r.URL.Path, status)
	if _, err := os.Stat(templatePath(file)); err != nil {
		http.Error(w, strconv.Itoa(status)+" — "+http.StatusText(status), status)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writePage(w, r, file, status) // pas de contrôle de méthode : un POST peut recevoir une 429
}

// Sous /en/, c'est la 404 anglaise qui s'affiche (si elle existe).
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	file := errorFile(r.URL.Path, http.StatusNotFound)
	w.Header().Set("Cache-Control", "no-store")
	renderPage(w, r, file, http.StatusNotFound)
}

func isEnglishPath(p string) bool {
	return p == "/en" || strings.HasPrefix(p, "/en/")
}
