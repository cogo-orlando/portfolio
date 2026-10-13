package middleware

// ══════════════════════════════════════════
//  PAGES D'ERREUR
//
//  Les middlewares (rate limit, honeypot, Cloudflare, panic…) répondent avec
//  une vraie page HTML au design du site, au lieu d'un texte brut.
//  Le rendu est fait par le package handler : server.go branche les deux
//  fonctions ci-dessous au démarrage (pas d'import circulaire).
//  Sans branchement (tests, outils), on retombe sur une réponse texte.
// ══════════════════════════════════════════

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ErrorPage rend la page d'erreur du code HTTP donné (403, 405, 429, 500, 503)
var ErrorPage func(w http.ResponseWriter, r *http.Request, status int)

// NotFoundPage rend la page 404 habituelle du site
var NotFoundPage func(w http.ResponseWriter, r *http.Request)

// writeError envoie la page d'erreur, avec les en-têtes de sécurité
// (certaines erreurs partent AVANT SecurityMiddleware dans la chaîne)
func writeError(w http.ResponseWriter, r *http.Request, status int) {
	setSecurityHeaders(w.Header())
	w.Header().Set("Cache-Control", "no-store") // une erreur ne doit jamais rester en cache
	if ErrorPage != nil {
		ErrorPage(w, r, status)
		return
	}
	http.Error(w, strconv.Itoa(status)+" — "+http.StatusText(status), status)
}

// writeNotFound : même 404 que n'importe quelle page inexistante.
// Pour le honeypot c'est important : un scanner ne doit pas pouvoir
// distinguer une route piège d'une vraie page absente.
func writeNotFound(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w.Header())
	if NotFoundPage != nil {
		NotFoundPage(w, r)
		return
	}
	http.NotFound(w, r)
}

// isStaticAsset : fichiers nécessaires à l'affichage des pages d'erreur.
// Ils ne passent pas par le rate limit, la blacklist ni la maintenance :
// sinon la page 429 ou 403 s'afficherait sans CSS ni polices.
// (Ils sont légers et mis en cache par Cloudflare pendant un an.)
func isStaticAsset(path string) bool {
	for _, p := range []string{"/css/", "/js/", "/fonts/", "/img/"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return path == "/favicon.ico"
}

// ══════════════════════════════════════════
//  MODE MAINTENANCE
//  MAINTENANCE=true dans les variables d'environnement Render
//  → toutes les pages renvoient la page 503 « Maintenance en cours ».
//  Restent accessibles : /health (health check Render), /api/status et les fichiers statiques.
// ══════════════════════════════════════════

func maintenanceEnabled() bool { return os.Getenv("MAINTENANCE") == "true" }

func MaintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if maintenanceEnabled() && p != "/health" && p != "/api/status" && !isStaticAsset(p) {
			w.Header().Set("Retry-After", "3600")
			writeError(w, r, http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}
