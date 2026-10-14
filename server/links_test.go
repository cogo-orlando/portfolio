package server

// ══════════════════════════════════════════
//  LIENS MORTS
//  Ouvre chaque page du site (FR et EN), récupère tous les liens internes
//  (href et src qui commencent par "/") et vérifie que chacun répond.
//  Une page supprimée mais encore liée quelque part fait échouer ce test.
// ══════════════════════════════════════════

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var linkRe = regexp.MustCompile(`(?:href|src)="(/[^"#]*)`)

func chdirRepoRoot(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod introuvable")
		}
		dir = parent
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestNoDeadInternalLinks(t *testing.T) {
	chdirRepoRoot(t)
	mux := newMux()

	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	// Toutes les pages : françaises + anglaises traduites
	var pages []string
	for _, p := range translatablePages() {
		pages = append(pages, p.path)
		if translated[p.path] {
			pages = append(pages, enPath(p.path))
		}
	}

	// lien → page où on l'a trouvé (pour un message d'erreur utile)
	links := map[string]string{}
	for _, page := range pages {
		w := get(page)
		if w.Code != http.StatusOK {
			t.Errorf("la page %s répond %d", page, w.Code)
			continue
		}
		for _, m := range linkRe.FindAllStringSubmatch(w.Body.String(), -1) {
			link := m[1]
			if strings.HasPrefix(link, "//") { // lien externe sans protocole
				continue
			}
			if _, seen := links[link]; !seen {
				links[link] = page
			}
		}
	}

	if len(links) == 0 {
		t.Fatal("aucun lien trouvé : le test ne vérifie rien")
	}

	keys := make([]string, 0, len(links))
	for k := range links {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, link := range keys {
		w := get(link)
		// 200 = OK ; 301/302 = redirection voulue (page anglaise pas encore traduite)
		if w.Code >= 400 {
			t.Errorf("lien mort : %s (trouvé sur %s) → %d", link, links[link], w.Code)
		}
	}
	t.Logf("%d pages ouvertes, %d liens internes vérifiés", len(pages), len(keys))
}
