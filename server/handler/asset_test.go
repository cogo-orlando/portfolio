package handler

import (
	"strings"
	"testing"
)

// Un chemin avec ".." est refusé et renvoyé tel quel (pas de lecture hors de web/)
func TestAssetURL_RejectsTraversal(t *testing.T) {
	got := assetURL("/../../etc/passwd")
	if strings.Contains(got, "?v=") {
		t.Fatalf("un chemin avec .. ne doit pas être versionné, obtenu %q", got)
	}
}

// Un fichier absent ne fait pas planter la page : chemin renvoyé sans version
func TestAssetURL_MissingFile(t *testing.T) {
	got := assetURL("/css/nexiste-pas.css")
	if got != "/css/nexiste-pas.css" {
		t.Fatalf("attendu le chemin brut, obtenu %q", got)
	}
}

// La fonction est bien disponible dans les templates
func TestTemplateFuncs_HasAsset(t *testing.T) {
	if _, ok := templateFuncs["asset"]; !ok {
		t.Fatal(`la fonction "asset" doit être enregistrée dans templateFuncs`)
	}
}