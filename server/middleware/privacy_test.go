package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnonymizeIP(t *testing.T) {
	cases := map[string]string{
		"203.0.113.42":             "203.0.113.0",
		" 198.51.100.7 ":           "198.51.100.0",
		"2001:db8:abcd:12:1:2:3:4": "2001:db8:abcd::",
		"::ffff:203.0.113.42":      "203.0.113.0",
		"pas-une-ip":               "unknown",
		"":                         "unknown",
	}
	for in, want := range cases {
		if got := AnonymizeIP(in); got != want {
			t.Errorf("AnonymizeIP(%q) : attendu %q, obtenu %q", in, want, got)
		}
	}
}

// Le journal des visites ne doit jamais contenir l'IP complète d'un visiteur
func TestLogger_AnonymizesVisitorIP(t *testing.T) {
	logs := captureLogs(t)
	w := httptest.NewRecorder()
	LoggerMiddleware(okHandler()).ServeHTTP(w, newRequest(http.MethodGet, "/home", "203.0.113.42"))

	out := logs.String()
	if strings.Contains(out, "203.0.113.42") {
		t.Errorf("l'IP complète ne doit pas apparaître dans le journal : %s", out)
	}
	if !strings.Contains(out, "203.0.113.0") {
		t.Errorf("l'IP anonymisée doit apparaître dans le journal : %s", out)
	}
}
