package server

import (
	"strings"
	"testing"
)

func TestI18n_EnPath(t *testing.T) {
	cases := map[string]string{"/": "/en", "/home": "/en/home", "/projects/zoo": "/en/projects/zoo"}
	for in, want := range cases {
		if got := enPath(in); got != want {
			t.Errorf("enPath(%q) : attendu %q, obtenu %q", in, want, got)
		}
	}
}

func TestI18n_FrPath(t *testing.T) {
	cases := map[string]string{"/en": "/", "/en/": "/", "/en/home": "/home", "/en/projects/zoo": "/projects/zoo"}
	for in, want := range cases {
		got, ok := frPath(in)
		if !ok || got != want {
			t.Errorf("frPath(%q) : attendu %q, obtenu %q (%v)", in, want, got, ok)
		}
	}
	for _, notEN := range []string{"/home", "/english", "/enx/home"} {
		if _, ok := frPath(notEN); ok {
			t.Errorf("%q n'est pas une page anglaise", notEN)
		}
	}
}

// Page anglaise pas encore traduite → redirection vers la française
func TestI18n_FallbackToFrench(t *testing.T) {
	if got, ok := fallbackTarget("/en/home"); !ok || got != "/home" {
		t.Errorf("/en/home : attendu /home, obtenu %q (%v)", got, ok)
	}
	if got, ok := fallbackTarget("/en/"); !ok || got != "/" {
		t.Errorf("/en/ : attendu /, obtenu %q (%v)", got, ok)
	}
}

// Une page qui n'existe dans aucune langue reste une 404 (pas de redirection ouverte)
func TestI18n_NoFallbackForUnknownPages(t *testing.T) {
	for _, p := range []string{"/en/nexiste-pas", "/en//evil.com", "/en/../etc/passwd"} {
		if got, ok := fallbackTarget(p); ok {
			t.Errorf("%q ne doit pas être redirigé (obtenu %q)", p, got)
		}
	}
}

func TestI18n_AlternatesHeader(t *testing.T) {
	h := alternatesHeader("/home")
	for _, want := range []string{
		`<https://orlandocogo.com/home>; rel="alternate"; hreflang="fr"`,
		`<https://orlandocogo.com/en/home>; rel="alternate"; hreflang="en"`,
		`hreflang="x-default"`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("en-tête Link : %q manquant dans %q", want, h)
		}
	}
}
