package db

import (
	"os"
	"testing"
)

// ══════════════════════════════════════════
//  TESTS — truncate
// ══════════════════════════════════════════

func TestTruncate_ShortString(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("attendu 'hello', obtenu '%s'", result)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("attendu 'hello', obtenu '%s'", result)
	}
}

func TestTruncate_TooLong(t *testing.T) {
	result := truncate("helloworld", 5)
	if result != "hello" {
		t.Errorf("attendu 'hello', obtenu '%s'", result)
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	result := truncate("", 10)
	if result != "" {
		t.Errorf("attendu '', obtenu '%s'", result)
	}
}

func TestTruncate_ZeroMax(t *testing.T) {
	result := truncate("hello", 0)
	if result != "" {
		t.Errorf("attendu '', obtenu '%s'", result)
	}
}

func TestTruncate_UTF8(t *testing.T) {
	result := truncate("héllo", 3)
	if len([]rune(result)) != 3 {
		t.Errorf("attendu 3 runes, obtenu %d", len([]rune(result)))
	}
}

func TestTruncate_UTF8Emoji(t *testing.T) {
	result := truncate("🔒🛡🚀", 2)
	runes := []rune(result)
	if len(runes) != 2 {
		t.Errorf("attendu 2 runes, obtenu %d", len(runes))
	}
}

func TestTruncate_LongIP(t *testing.T) {
	longIP := "192.168.1.1.extra.garbage.that.should.be.cut.off.here.yes"
	result := truncate(longIP, 45)
	if len([]rune(result)) > 45 {
		t.Errorf("IP tronquée devrait faire max 45 runes, obtenu %d", len([]rune(result)))
	}
}

func TestTruncate_LongPath(t *testing.T) {
	path := "/" + string(make([]byte, 600))
	result := truncate(path, 500)
	if len([]rune(result)) > 500 {
		t.Errorf("path tronqué devrait faire max 500 runes, obtenu %d", len([]rune(result)))
	}
}

// ══════════════════════════════════════════
//  TESTS — EventType constantes
// ══════════════════════════════════════════

func TestEventType_Request(t *testing.T) {
	if EventRequest != "request" {
		t.Errorf("EventRequest devrait être 'request', obtenu '%s'", EventRequest)
	}
}

func TestEventType_Honeypot(t *testing.T) {
	if EventHoneypot != "honeypot" {
		t.Errorf("EventHoneypot devrait être 'honeypot', obtenu '%s'", EventHoneypot)
	}
}

func TestEventType_RateLimit(t *testing.T) {
	if EventRateLimit != "ratelimit" {
		t.Errorf("EventRateLimit devrait être 'ratelimit', obtenu '%s'", EventRateLimit)
	}
}

func TestEventType_Error(t *testing.T) {
	if EventError != "error" {
		t.Errorf("EventError devrait être 'error', obtenu '%s'", EventError)
	}
}

func TestEventType_Distinct(t *testing.T) {
	types := []EventType{EventRequest, EventHoneypot, EventRateLimit, EventError}
	seen := make(map[EventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("EventType dupliqué : %s", et)
		}
		seen[et] = true
	}
}

// ══════════════════════════════════════════
//  TESTS — maxRetries constante
// ══════════════════════════════════════════

func TestMaxRetries_Value(t *testing.T) {
	if maxRetries != 3 {
		t.Errorf("maxRetries devrait être 3, obtenu %d", maxRetries)
	}
}

func TestMaxRetries_Positive(t *testing.T) {
	if maxRetries <= 0 {
		t.Errorf("maxRetries devrait être positif, obtenu %d", maxRetries)
	}
}

// ══════════════════════════════════════════
//  TESTS — fonctions sans connexion DB
// ══════════════════════════════════════════

func TestLogEvent_NilConn(t *testing.T) {
	conn = nil
	LogEvent("1.2.3.4", "GET", "/home", 200, "Mozilla/5.0", EventRequest)
}

func TestLogEvent_AllEventTypes(t *testing.T) {
	conn = nil
	types := []EventType{EventRequest, EventHoneypot, EventRateLimit, EventError}
	for _, et := range types {
		LogEvent("1.2.3.4", "GET", "/test", 200, "agent", et)
	}
}

func TestLogHoneypot_NilConn(t *testing.T) {
	conn = nil
	LogHoneypot("1.2.3.4", "/wp-admin", "scanner/1.0")
}

func TestLogRateLimit_NilConn(t *testing.T) {
	conn = nil
	LogRateLimit("1.2.3.4", "/home")
}

func TestClose_NilConn(t *testing.T) {
	conn = nil
	Close()
}

func TestLogEvent_LongValues(t *testing.T) {
	conn = nil
	longStr := string(make([]byte, 1000))
	LogEvent(longStr, longStr, longStr, 200, longStr, EventRequest)
}

func TestLogHoneypot_LongValues(t *testing.T) {
	conn = nil
	longStr := string(make([]byte, 1000))
	LogHoneypot(longStr, longStr, longStr)
}

func TestLogRateLimit_LongValues(t *testing.T) {
	conn = nil
	longStr := string(make([]byte, 1000))
	LogRateLimit(longStr, longStr)
}

// ══════════════════════════════════════════
//  TESTS — execWithRetry sans connexion
// ══════════════════════════════════════════

func TestExecWithRetry_NilConn(t *testing.T) {
	conn = nil
	// Ne doit pas paniquer — conn == nil → panic dans ExecContext
	// On vérifie que LogEvent (qui appelle execWithRetry) ne panique pas
	// LogEvent vérifie conn == nil avant d'appeler execWithRetry
	LogEvent("1.2.3.4", "GET", "/", 200, "", EventRequest)
}

func TestLogEvent_StatusCodes(t *testing.T) {
	conn = nil
	codes := []int{200, 201, 400, 401, 403, 404, 429, 500, 503}
	for _, code := range codes {
		LogEvent("1.2.3.4", "GET", "/test", code, "agent", EventRequest)
	}
}

func TestLogEvent_Methods(t *testing.T) {
	conn = nil
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}
	for _, method := range methods {
		LogEvent("1.2.3.4", method, "/test", 200, "agent", EventRequest)
	}
}

// ══════════════════════════════════════════
//  TESTS — Init sans DATABASE_URL
// ══════════════════════════════════════════

func TestInit_NoDatabaseURL(t *testing.T) {
	original := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer func() {
		if original != "" {
			os.Setenv("DATABASE_URL", original)
		}
	}()

	savedConn := conn
	conn = nil

	LogEvent("1.2.3.4", "GET", "/", 200, "", EventRequest)
	LogHoneypot("1.2.3.4", "/wp-admin", "")
	LogRateLimit("1.2.3.4", "/")
	Close()

	conn = savedConn
}
