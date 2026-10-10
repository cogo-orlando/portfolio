// Package db enregistre les événements de sécurité du portfolio dans Supabase (PostgreSQL).
//
// Principes :
//   - Jamais bloquant : les événements passent par une file d'attente bornée,
//     écrite en base par un seul worker, par lots. Si la base est lente ou
//     si le site est inondé de requêtes, les événements en trop sont ignorés
//     (et comptés) au lieu de créer des milliers de goroutines.
//   - RGPD : l'IP des visiteurs "normaux" est anonymisée avant stockage,
//     et toutes les données sont supprimées automatiquement après N jours.
//   - Désactivé proprement si DATABASE_URL n'est pas définie (dev, tests).
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq"
)

// ══════════════════════════════════════════
//  CONFIGURATION
// ══════════════════════════════════════════

const (
	queueSize            = 1000            // événements max en attente
	batchSize            = 100             // événements max par INSERT
	flushInterval        = 2 * time.Second // écriture au moins toutes les 2 s
	retentionInterval    = 24 * time.Hour  // nettoyage une fois par jour
	defaultRetentionDays = 30
	maxRetries           = 3
)

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

type event struct {
	ip        string
	method    string
	path      string
	status    int
	userAgent string
	eventType EventType
	blacklist bool // honeypot : ajouter aussi l'IP à blacklisted_ips
}

// ══════════════════════════════════════════
//  ÉTAT
// ══════════════════════════════════════════

var (
	conn *sql.DB
	once sync.Once

	queue   chan event
	qMu     sync.RWMutex // protège queue/closed (pas d'envoi après Close)
	closed  bool
	worker  sync.WaitGroup
	dropped atomic.Int64 // événements ignorés car la file était pleine

	retentionDays = defaultRetentionDays
)

// ══════════════════════════════════════════
//  CONNEXION
// ══════════════════════════════════════════

func Init() {
	once.Do(func() {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			slog.Warn("DATABASE_URL non définie — logs DB désactivés")
			return
		}
		dsn = requireSSL(dsn)

		if v, err := strconv.Atoi(os.Getenv("LOG_RETENTION_DAYS")); err == nil && v > 0 {
			retentionDays = v
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
			_ = c.Close()
			return
		}

		c.SetMaxOpenConns(5)
		c.SetMaxIdleConns(2)
		c.SetConnMaxLifetime(5 * time.Minute)
		c.SetConnMaxIdleTime(2 * time.Minute)
		conn = c

		qMu.Lock()
		queue = make(chan event, queueSize)
		closed = false
		qMu.Unlock()

		worker.Add(1)
		go runWorker(queue)

		slog.Info("connexion Supabase établie — logging activé",
			"retention_days", retentionDays,
			"queue_size", queueSize,
		)
	})
}

// Close vide la file d'attente (les derniers événements sont écrits), puis ferme la connexion.
// À appeler à l'arrêt du serveur.
func Close() {
	qMu.Lock()
	if queue != nil && !closed {
		closed = true
		close(queue)
	}
	qMu.Unlock()

	worker.Wait()

	if conn != nil {
		_ = conn.Close() // #nosec G104
		conn = nil
	}
}

// Supabase exige une connexion chiffrée : si le DSN ne précise rien, on force TLS.
func requireSSL(dsn string) string {
	if strings.Contains(dsn, "sslmode=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "sslmode=require"
}

// ══════════════════════════════════════════
//  INFOS (pour /health ou les logs)
// ══════════════════════════════════════════

// Enabled indique si l'enregistrement en base est actif
func Enabled() bool { return conn != nil }

// Stats : état de la file d'attente
type Stats struct {
	Enabled       bool  `json:"enabled"`
	Queued        int   `json:"queued"`
	Dropped       int64 `json:"dropped"`
	RetentionDays int   `json:"retention_days"`
}

func GetStats() Stats {
	qMu.RLock()
	defer qMu.RUnlock()
	queued := 0
	if queue != nil && !closed {
		queued = len(queue)
	}
	return Stats{Enabled: Enabled(), Queued: queued, Dropped: dropped.Load(), RetentionDays: retentionDays}
}

// ══════════════════════════════════════════
//  API PUBLIQUE (appelée par le middleware)
// ══════════════════════════════════════════

// LogEvent enregistre une requête. Les fichiers statiques réussis (CSS, JS, images)
// ne sont pas stockés : ils n'apportent rien et rempliraient la base.
func LogEvent(ip, method, path string, status int, userAgent string, eventType EventType) {
	if !shouldStore(path, status, eventType) {
		return
	}
	enqueue(event{ip: ip, method: method, path: path, status: status, userAgent: userAgent, eventType: eventType})
}

// LogHoneypot enregistre un accès à une route piège et blackliste l'IP 24 h
func LogHoneypot(ip, path, userAgent string) {
	enqueue(event{ip: ip, method: "GET", path: path, status: 404, userAgent: userAgent, eventType: EventHoneypot, blacklist: true})
}

// LogRateLimit enregistre un dépassement du rate limit
func LogRateLimit(ip, path string) {
	enqueue(event{ip: ip, method: "GET", path: path, status: 429, eventType: EventRateLimit})
}

func shouldStore(path string, status int, eventType EventType) bool {
	if eventType != EventRequest || status >= 400 {
		return true
	}
	// Vérifications automatiques (healthcheck Docker, page Status) : aucun intérêt à stocker
	if path == "/health" || path == "/api/status" {
		return false
	}
	for _, prefix := range []string{"/css/", "/js/", "/img/", "/fonts/", "/favicon"} {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}

// enqueue n'attend jamais : si la file est pleine, l'événement est ignoré et compté
func enqueue(e event) {
	qMu.RLock()
	defer qMu.RUnlock()
	if queue == nil || closed {
		return
	}
	select {
	case queue <- prepare(e):
	default:
		if n := dropped.Add(1); n == 1 || n%1000 == 0 {
			slog.Warn("file d'événements DB pleine — événements ignorés", "dropped_total", n)
		}
	}
}

// prepare nettoie et anonymise un événement avant stockage
func prepare(e event) event {
	ip := e.ip
	// Visiteurs normaux : IP anonymisée (RGPD).
	// Honeypot / rate limit : IP complète, nécessaire pour la sécurité (intérêt légitime),
	// supprimée comme le reste après la durée de conservation.
	if e.eventType == EventRequest || e.eventType == EventError {
		ip = anonymizeIP(ip)
	}
	e.ip = truncate(sanitize(ip), 45)
	e.method = truncate(sanitize(e.method), 10)
	e.path = truncate(sanitize(e.path), 500)
	e.userAgent = truncate(sanitize(e.userAgent), 512)
	return e
}

// ══════════════════════════════════════════
//  WORKER : écrit les événements par lots
// ══════════════════════════════════════════

func runWorker(q <-chan event) {
	defer worker.Done()

	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	retentionTicker := time.NewTicker(retentionInterval)
	defer retentionTicker.Stop()

	cleanup() // un premier nettoyage au démarrage

	batch := make([]event, 0, batchSize)
	for {
		select {
		case e, ok := <-q:
			if !ok { // file fermée par Close() : on écrit ce qui reste et on s'arrête
				flush(batch)
				return
			}
			batch = append(batch, e)
			if len(batch) >= batchSize {
				flush(batch)
				batch = batch[:0]
			}
		case <-flushTicker.C:
			if len(batch) > 0 {
				flush(batch)
				batch = batch[:0]
			}
		case <-retentionTicker.C:
			cleanup()
		}
	}
}

// flush insère un lot en UNE requête (beaucoup plus léger qu'un INSERT par événement)
func flush(batch []event) {
	if len(batch) == 0 || conn == nil {
		return
	}
	query, args := buildInsert(batch)
	_ = execWithRetry(query, args...)

	for _, e := range batch {
		if !e.blacklist {
			continue
		}
		if err := execWithRetry(`
			INSERT INTO blacklisted_ips (ip, reason, expires_at)
			VALUES ($1, 'honeypot', $2)
			ON CONFLICT (ip) DO UPDATE SET expires_at = EXCLUDED.expires_at
		`, e.ip, time.Now().Add(24*time.Hour)); err != nil {
			slog.Error("erreur insert blacklist", "error", err, "ip", e.ip)
		}
	}
}

// buildInsert construit un INSERT multi-lignes paramétré (aucune concaténation de données)
func buildInsert(batch []event) (string, []interface{}) {
	var sb strings.Builder
	sb.WriteString("INSERT INTO security_events (ip, method, path, status, user_agent, event_type) VALUES ")
	args := make([]interface{}, 0, len(batch)*6)
	for i, e := range batch {
		if i > 0 {
			sb.WriteString(", ")
		}
		n := i * 6
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6)
		args = append(args, e.ip, e.method, e.path, e.status, e.userAgent, string(e.eventType))
	}
	return sb.String(), args
}

// cleanup applique la durée de conservation (RGPD) et purge les blacklists expirées
func cleanup() {
	if conn == nil {
		return
	}
	if err := execWithRetry(
		`DELETE FROM security_events WHERE created_at < now() - make_interval(days => $1)`,
		retentionDays,
	); err == nil {
		slog.Info("rétention appliquée", "older_than_days", retentionDays)
	}
	_ = execWithRetry(`DELETE FROM blacklisted_ips WHERE expires_at < now()`)
}

// ══════════════════════════════════════════
//  RETRY AVEC BACKOFF EXPONENTIEL
//  1ère retry : 100ms · 2ème : 200ms
// ══════════════════════════════════════════

func execWithRetry(query string, args ...interface{}) error {
	if conn == nil {
		return fmt.Errorf("db non initialisée")
	}
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
		slog.Warn("db.exec retry", "attempt", attempt+1, "max", maxRetries, "error", err)
	}

	slog.Error("db.exec échec après retries", "attempts", maxRetries, "error", lastErr)
	return lastErr
}

// ══════════════════════════════════════════
//  HELPERS
// ══════════════════════════════════════════

// anonymizeIP retire la partie qui identifie la machine :
// IPv4 → dernier octet à 0 (203.0.113.42 → 203.0.113.0)
// IPv6 → on garde le préfixe /48 (2001:db8:1234:5678::1 → 2001:db8:1234::)
func anonymizeIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return "unknown"
	}
	if v4 := ip.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return ip.Mask(net.CIDRMask(48, 128)).String()
}

// sanitize rend une chaîne acceptable par PostgreSQL :
// UTF-8 valide (sinon l'INSERT échoue) et sans octet NUL (refusé dans le type text).
// Utile car user-agent et chemins viennent directement des visiteurs… et des attaquants.
func sanitize(s string) string {
	s = strings.ToValidUTF8(s, "�")
	return strings.ReplaceAll(s, "\x00", "")
}

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