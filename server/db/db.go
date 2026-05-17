package db

import (
	"context"
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.PingContext(ctx); err != nil {
			slog.Error("db.Ping failed", "error", err)
			return
		}

		c.SetMaxOpenConns(5)
		c.SetMaxIdleConns(2)
		c.SetConnMaxLifetime(5 * time.Minute)
		c.SetConnMaxIdleTime(2 * time.Minute)

		conn = c
		slog.Info("connexion Supabase établie — logging activé")
	})
}

func Close() {
	if conn != nil {
		_ = conn.Close() // #nosec G104
	}
}

// ══════════════════════════════════════════
//  TYPES
// ══════════════════════════════════════════

type EventType string

const (
	EventRequest   EventType = "request"
	EventHoneypot  EventType = "honeypot"
	EventRateLimit EventType = "ratelimit"
	EventError     EventType = "error"
)

// ══════════════════════════════════════════
//  RETRY AVEC BACKOFF EXPONENTIEL
//  Tente maxRetries fois avec délai croissant
//  1ère retry : 100ms · 2ème : 200ms · 3ème : 400ms
// ══════════════════════════════════════════

const maxRetries = 3

func execWithRetry(query string, args ...interface{}) error {
	var lastErr error
	delay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := conn.ExecContext(ctx, query, args...)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err
		slog.Warn("db.exec retry",
			"attempt", attempt+1,
			"max", maxRetries,
			"error", err,
		)
	}

	slog.Error("db.exec échec après retries",
		"attempts", maxRetries,
		"error", lastErr,
	)
	return lastErr
}

// ══════════════════════════════════════════
//  LOGGING ÉVÉNEMENTS
// ══════════════════════════════════════════

func LogEvent(ip, method, path string, status int, userAgent string, eventType EventType) {
	if conn == nil {
		return
	}

	ip = truncate(ip, 45)
	method = truncate(method, 10)
	path = truncate(path, 500)
	userAgent = truncate(userAgent, 512)
	et := string(eventType)

	go func() {
		_ = execWithRetry(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, ip, method, path, status, userAgent, et)
	}()
}

func LogHoneypot(ip, path, userAgent string) {
	if conn == nil {
		return
	}

	ip = truncate(ip, 45)
	path = truncate(path, 500)
	userAgent = truncate(userAgent, 512)
	expires := time.Now().Add(24 * time.Hour)

	go func() {
		_ = execWithRetry(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, 'GET', $2, 404, $3, 'honeypot')
		`, ip, path, userAgent)

		_ = execWithRetry(`
			INSERT INTO blacklisted_ips (ip, reason, expires_at)
			VALUES ($1, 'honeypot', $2)
			ON CONFLICT (ip) DO UPDATE SET expires_at = $2
		`, ip, expires)
	}()
}

func LogRateLimit(ip, path string) {
	if conn == nil {
		return
	}

	ip = truncate(ip, 45)
	path = truncate(path, 500)

	go func() {
		_ = execWithRetry(`
			INSERT INTO security_events (ip, method, path, status, user_agent, event_type)
			VALUES ($1, 'GET', $2, 429, '', 'ratelimit')
		`, ip, path)
	}()
}

// ══════════════════════════════════════════
//  HELPERS
// ══════════════════════════════════════════

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
