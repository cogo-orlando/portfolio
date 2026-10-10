package db

import (
	"strings"
	"testing"
)

func TestPrivacy_AnonymizeIPv4(t *testing.T) {
	if got := anonymizeIP("203.0.113.42"); got != "203.0.113.0" {
		t.Errorf("attendu 203.0.113.0, obtenu %s", got)
	}
}

func TestPrivacy_AnonymizeIPv6(t *testing.T) {
	if got := anonymizeIP("2001:db8:1234:5678::1"); got != "2001:db8:1234::" {
		t.Errorf("attendu 2001:db8:1234::, obtenu %s", got)
	}
}

func TestPrivacy_AnonymizeInvalid(t *testing.T) {
	for _, raw := range []string{"", "pas-une-ip", "1.2.3.4:5678"} {
		if got := anonymizeIP(raw); got != "unknown" {
			t.Errorf("%q : attendu unknown, obtenu %s", raw, got)
		}
	}
}

// Visiteur normal → IP anonymisée ; attaque (honeypot, rate limit) → IP complète
func TestPrivacy_PrepareAnonymizesOnlyNormalTraffic(t *testing.T) {
	if e := prepare(event{ip: "198.51.100.7", eventType: EventRequest}); e.ip != "198.51.100.0" {
		t.Errorf("requête normale : IP attendue anonymisée, obtenu %s", e.ip)
	}
	if e := prepare(event{ip: "198.51.100.7", eventType: EventError}); e.ip != "198.51.100.0" {
		t.Errorf("erreur : IP attendue anonymisée, obtenu %s", e.ip)
	}
	if e := prepare(event{ip: "198.51.100.7", eventType: EventHoneypot}); e.ip != "198.51.100.7" {
		t.Errorf("honeypot : IP complète attendue, obtenu %s", e.ip)
	}
	if e := prepare(event{ip: "198.51.100.7", eventType: EventRateLimit}); e.ip != "198.51.100.7" {
		t.Errorf("rate limit : IP complète attendue, obtenu %s", e.ip)
	}
}

// PostgreSQL refuse l'UTF-8 invalide et l'octet NUL : on nettoie avant d'insérer
func TestSanitize_InvalidUTF8AndNUL(t *testing.T) {
	got := sanitize("curl\x00/7.0 \xff\xfe")
	if strings.Contains(got, "\x00") {
		t.Error("l'octet NUL doit être retiré")
	}
	if !strings.HasPrefix(got, "curl/7.0") {
		t.Errorf("contenu inattendu : %q", got)
	}
	for _, r := range got {
		if r == 0xFFFD {
			return // les octets invalides ont été remplacés
		}
	}
	t.Error("les octets UTF-8 invalides doivent être remplacés")
}

func TestPrepare_TruncatesLongFields(t *testing.T) {
	e := prepare(event{ip: "1.2.3.4", path: "/" + strings.Repeat("a", 2000), userAgent: strings.Repeat("b", 2000), eventType: EventHoneypot})
	if len([]rune(e.path)) != 500 || len([]rune(e.userAgent)) != 512 {
		t.Errorf("longueurs attendues 500/512, obtenu %d/%d", len([]rune(e.path)), len([]rune(e.userAgent)))
	}
}

func TestShouldStore_SkipsSuccessfulStaticFiles(t *testing.T) {
	cases := []struct {
		path   string
		status int
		et     EventType
		want   bool
	}{
		{"/css/nav.css", 200, EventRequest, false},
		{"/js/core.js", 304, EventRequest, false},
		{"/img/favicon.png", 200, EventRequest, false},
		{"/css/absent.css", 404, EventError, true}, // une erreur est toujours gardée
		{"/home", 200, EventRequest, true},
		{"/health", 200, EventRequest, false},     // healthcheck Docker
		{"/api/status", 200, EventRequest, false}, // page Status
		{"/health", 500, EventError, true},        // mais une panne est gardée
		{"/wp-admin", 404, EventHoneypot, true},
	}
	for _, c := range cases {
		if got := shouldStore(c.path, c.status, c.et); got != c.want {
			t.Errorf("%s (%d, %s) : attendu %v, obtenu %v", c.path, c.status, c.et, c.want, got)
		}
	}
}

// Sans DATABASE_URL, rien ne doit planter ni bloquer
func TestDisabled_NoopWithoutConnection(t *testing.T) {
	LogEvent("1.2.3.4", "GET", "/home", 200, "test", EventRequest)
	LogHoneypot("1.2.3.4", "/wp-admin", "test")
	LogRateLimit("1.2.3.4", "/")
	if s := GetStats(); s.Enabled || s.Queued != 0 {
		t.Errorf("base désactivée attendue, obtenu %+v", s)
	}
}

// File pleine → l'événement est ignoré et compté, jamais bloquant
func TestEnqueue_DropsWhenFull(t *testing.T) {
	qMu.Lock()
	previous, prevClosed := queue, closed
	queue, closed = make(chan event, 1), false
	qMu.Unlock()
	before := dropped.Load()
	t.Cleanup(func() {
		qMu.Lock()
		queue, closed = previous, prevClosed
		qMu.Unlock()
	})

	LogRateLimit("1.2.3.4", "/") // remplit la file
	LogRateLimit("1.2.3.4", "/") // file pleine → ignoré

	if got := dropped.Load() - before; got != 1 {
		t.Errorf("1 événement ignoré attendu, obtenu %d", got)
	}
}

// L'INSERT par lot est entièrement paramétré : aucune donnée dans le SQL
func TestBuildInsert_Parameterized(t *testing.T) {
	batch := []event{
		{ip: "1.1.1.0", method: "GET", path: "/'; DROP TABLE security_events;--", status: 200, eventType: EventRequest},
		{ip: "2.2.2.2", method: "GET", path: "/wp-admin", status: 404, eventType: EventHoneypot},
	}
	query, args := buildInsert(batch)
	if strings.Contains(query, "DROP TABLE") {
		t.Fatal("les données ne doivent jamais être concaténées dans la requête SQL")
	}
	if !strings.Contains(query, "($7, $8, $9, $10, $11, $12)") {
		t.Errorf("placeholders inattendus : %s", query)
	}
	if len(args) != 12 {
		t.Errorf("12 arguments attendus, obtenu %d", len(args))
	}
}

func TestRequireSSL(t *testing.T) {
	cases := map[string]string{
		"postgres://u:p@h/db":                     "postgres://u:p@h/db?sslmode=require",
		"postgres://u:p@h/db?connect_timeout=5":   "postgres://u:p@h/db?connect_timeout=5&sslmode=require",
		"postgres://u:p@h/db?sslmode=verify-full": "postgres://u:p@h/db?sslmode=verify-full",
	}
	for in, want := range cases {
		if got := requireSSL(in); got != want {
			t.Errorf("%s : attendu %s, obtenu %s", in, want, got)
		}
	}
}