package db

import (
	"database/sql"
	"log/slog"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// ══════════════════════════════════════════
//  CONNEXION
// ══════════════════════════════════════════

var (
	conn *sql.DB
	once sync.Once
)

// Init ouvre la connexion à Supabase (appelé une seule fois au démarrage)
func Init() {
	once.Do(func() {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			slog.Warn("DATABASE_URL non définie — logs DB désactivés")
			return
		}

		c, err := sql.Open("postgres", dsn)
		if err != nil {
			slog.Error("db.Open failed", "error", err)
			return
		}

		if err := c.Ping(); err != nil {
			slog.Error("db.Ping failed", "error", err)
			return
		}

		c.SetMaxOpenConns(5)
		c.SetMaxIdleConns(2)
		c.SetConnMaxLifetime(5 * time.Minute)

		conn = c
		slog.Info("connexion Supabase établie — logging activé")
	})
}

// ══════════════════════════════════════════
//  LOGGING ÉVÉNEMENTS
// ══════════════════════════════════════════

type EventType string

const (
	EventRequest   EventType = "request"
	EventHoneypot  EventType = "honeypot"
	EventRateLimit EventType = "ratelimit"
	EventError     EventType = "error"
)

// LogEvent insère un événement en DB de manière asynchrone
func LogEvent(ip, method, path string, status int, userAgent string, eventType EventType) {
	if conn == nil {
		return
	}

	go func() {
		_, err := conn.Exec(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, ip, method, path, status, userAgent, string(eventType))

		if err != nil {
			slog.Error("db.LogEvent failed", "error", err)
		}
	}()
}

// LogHoneypot insère un événement honeypot + blackliste l'IP
func LogHoneypot(ip, path, userAgent string) {
	if conn == nil {
		return
	}

	go func() {
		conn.Exec(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, 'GET', $2, 404, $3, 'honeypot')
		`, ip, path, userAgent) //nolint:errcheck

		conn.Exec(`
			INSERT INTO blacklisted_ips (ip, reason, expires_at)
			VALUES ($1, 'honeypot', $2)
			ON CONFLICT (ip) DO UPDATE SET expires_at = $2
		`, ip, time.Now().Add(24*time.Hour)) //nolint:errcheck
	}()
}

// LogRateLimit insère un événement rate limit
func LogRateLimit(ip, path string) {
	if conn == nil {
		return
	}

	go func() {
		conn.Exec(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, 'GET', $2, 429, '', 'ratelimit')
		`, ip, path) //nolint:errcheck
	}()
}

// Close ferme la connexion DB
func Close() {
	if conn != nil {
		conn.Close()
	}
}
