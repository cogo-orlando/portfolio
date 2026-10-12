package server

// ══════════════════════════════════════════
//  VERSION ANGLAISE
//
//  - Les pages anglaises sont dans web/html/en/ (mêmes noms de fichiers qu'en français).
//  - Elles sont servies sous /en/... : /en/home, /en/projects/netflix…
//  - Traduction progressive : une page est en ligne en anglais dès que son
//    fichier existe. Tant qu'elle n'existe pas, /en/xxx redirige vers /xxx.
//  - Chaque page traduite annonce sa version dans l'autre langue (hreflang)
//    via l'en-tête HTTP Link : Google propose la bonne langue à chacun.
// ══════════════════════════════════════════

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"portfo/server/handler"
)

const (
	siteURL  = "https://orlandocogo.com"
	enPrefix = "/en"
)

// page : une route française et son template
type page struct {
	path string
	file string
}

// translatablePages : l'accueil + toutes les pages du sitemap
func translatablePages() []page {
	pages := []page{{"/", "index.html"}}
	for _, u := range sitemapURLs {
		pages = append(pages, page{u.path, u.file})
	}
	return pages
}

// enPath : "/home" → "/en/home", "/" → "/en"
func enPath(frPath string) string {
	if frPath == "/" {
		return enPrefix
	}
	return enPrefix + frPath
}

// frPath : "/en/home" → "/home", "/en" ou "/en/" → "/"
func frPath(path string) (string, bool) {
	if path == enPrefix || path == enPrefix+"/" {
		return "/", true
	}
	if rest, ok := strings.CutPrefix(path, enPrefix+"/"); ok {
		return "/" + rest, true
	}
	return "", false
}

func englishFileExists(file string) bool {
	_, err := os.Stat(filepath.Join("web", "html", "en", filepath.FromSlash(file)))
	return err == nil
}

// translated : routes françaises qui ont une version anglaise (calculé au démarrage)
var translated = map[string]bool{}

// registerEnglish ajoute les routes /en/... des pages déjà traduites
func registerEnglish(mux *http.ServeMux) {
	translated = map[string]bool{}
	for _, p := range translatablePages() {
		if !englishFileExists(p.file) {
			continue
		}
		translated[p.path] = true
		h := withAlternates(p.path, handler.PageHandler("en/"+p.file))
		mux.HandleFunc(enPath(p.path), h)
		if p.path == "/" {
			mux.HandleFunc(enPrefix+"/", func(w http.ResponseWriter, r *http.Request) {
				// "/en/" exactement → accueil anglais ; "/en/xxx" non traduit → page française
				if r.URL.Path == enPrefix+"/" {
					h(w, r)
					return
				}
				englishFallback(w, r)
			})
		}
	}
	// Accueil anglais pas encore traduit : /en/... est quand même géré (redirection)
	if !translated["/"] {
		mux.HandleFunc(enPrefix+"/", englishFallback)
		mux.HandleFunc(enPrefix, englishFallback)
	}
}

// englishFallback : page anglaise pas encore traduite → version française (302),
// sinon vraie 404
func englishFallback(w http.ResponseWriter, r *http.Request) {
	if target, ok := fallbackTarget(r.URL.Path); ok {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	handler.NotFoundHandler(w, r)
}

// fallbackTarget renvoie la page française correspondante, si elle existe
func fallbackTarget(path string) (string, bool) {
	fr, ok := frPath(path)
	if !ok {
		return "", false
	}
	if fr == "/" {
		return "/", true
	}
	if _, exists := routes[fr]; exists {
		return fr, true
	}
	return "", false
}

// withAlternates ajoute l'en-tête hreflang si la page existe dans les deux langues
func withAlternates(frPath string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if translated[frPath] {
			w.Header().Set("Link", alternatesHeader(frPath))
		}
		next(w, r)
	}
}

// alternatesHeader : <…/home>; rel="alternate"; hreflang="fr", <…/en/home>; …
func alternatesHeader(frPath string) string {
	fr := siteURL + frPath
	if frPath == "/" {
		fr = siteURL + "/"
	}
	en := siteURL + enPath(frPath)
	return `<` + fr + `>; rel="alternate"; hreflang="fr", ` +
		`<` + en + `>; rel="alternate"; hreflang="en", ` +
		`<` + fr + `>; rel="alternate"; hreflang="x-default"`
}
