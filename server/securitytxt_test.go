package server

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════
//  security.txt (RFC 9116)
//  Le champ Expires est obligatoire et doit être renouvelé (moins d'un an).
//  Ce test échoue 30 jours avant l'expiration : le scan CI hebdomadaire
//  te prévient à temps.
// ══════════════════════════════════════════

func securityTxtPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "web", ".well-known", "security.txt")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod introuvable")
		}
		dir = parent
	}
}

func readSecurityTxt(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(securityTxtPath(t))
	if err != nil {
		t.Fatalf("security.txt introuvable : %v", err)
	}
	defer f.Close()

	fields := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return fields
}

func TestSecurityTxt_RequiredFields(t *testing.T) {
	fields := readSecurityTxt(t)
	for _, k := range []string{"Contact", "Expires"} {
		if fields[k] == "" {
			t.Errorf("champ obligatoire manquant : %s", k)
		}
	}
	if c := fields["Contact"]; c != "" && !strings.HasPrefix(c, "mailto:") && !strings.HasPrefix(c, "https://") {
		t.Errorf("Contact doit commencer par mailto: ou https:// (obtenu %q)", c)
	}
}

func TestSecurityTxt_NotExpiringSoon(t *testing.T) {
	raw := readSecurityTxt(t)["Expires"]
	exp, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("Expires doit être au format RFC 3339 (ex. 2027-10-01T00:00:00.000Z) : %v", err)
	}
	now := time.Now()
	if exp.Before(now.Add(30 * 24 * time.Hour)) {
		t.Errorf("security.txt expire le %s : mets à jour le champ Expires (max 1 an)", exp.Format("02/01/2006"))
	}
	if exp.After(now.Add(366 * 24 * time.Hour)) {
		t.Errorf("Expires ne doit pas dépasser un an (RFC 9116) : %s", exp.Format("02/01/2006"))
	}
}
