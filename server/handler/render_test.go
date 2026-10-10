package handler

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ══════════════════════════════════════════
//  TEST DE RENDU : chaque page HTML doit s'afficher sans erreur.
//  Si un template est cassé (partial manquant, {{template}} inconnu,
//  fichier CSS/JS introuvable…), la CI échoue → pas de déploiement.
// ══════════════════════════════════════════

// Les tests tournent depuis server/handler : on se place à la racine du projet
// (là où se trouve go.mod) pour que "web/html" soit trouvé, puis on revient.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod introuvable : impossible de trouver la racine du projet")
		}
		dir = parent
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

// Toutes les pages HTML, sauf les partials
func listPages(t *testing.T) []string {
	t.Helper()
	var pages []string
	err := filepath.WalkDir(htmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "partials" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".html") {
			rel, err := filepath.Rel(htmlDir, path)
			if err != nil {
				return err
			}
			pages = append(pages, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("aucune page trouvée dans " + htmlDir)
	}
	return pages
}

// Liens vers des CSS/JS locaux dans le HTML rendu
var assetLinkRe = regexp.MustCompile(`(?:href|src)="(/(?:css|js)/[^"]+)"`)

func TestAllPagesRender(t *testing.T) {
	chdirToRepoRoot(t)

	for _, page := range listPages(t) {
		page := page
		t.Run(page, func(t *testing.T) {
			tmpl, err := parseTemplate(page)
			if err != nil {
				t.Fatalf("template invalide : %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, nil); err != nil {
				t.Fatalf("rendu impossible : %v", err)
			}
			html := buf.String()

			if !strings.Contains(html, "</html>") {
				t.Error("la page rendue est incomplète (pas de </html>)")
			}
			if strings.Contains(html, "<no value>") {
				t.Error(`la page contient "<no value>" : une donnée attendue par le template est absente`)
			}
			if strings.Contains(html, "{{") {
				t.Error(`la page contient "{{" : une balise de template n'a pas été interprétée`)
			}

			// Chaque CSS/JS doit passer par {{asset}} ET exister sur le disque
			// (asset() ne met pas de ?v= quand le fichier est introuvable)
			for _, m := range assetLinkRe.FindAllStringSubmatch(html, -1) {
				if !strings.Contains(m[1], "?v=") {
					t.Errorf("%s : fichier introuvable dans web/, ou lien écrit sans {{asset}}", m[1])
				}
			}
		})
	}
}
