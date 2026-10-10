// Package monitor vérifie régulièrement que chaque page du site répond,
// et expose le résultat en JSON pour la page Status.
//
// Les vérifications se font EN INTERNE (appel direct au routeur Go) :
// pas de trafic réseau, pas de passage par le rate limit ni par les logs.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// Target : une route à surveiller
type Target struct {
	Name string
	Path string
}

// RouteStatus : état d'une route, tel qu'envoyé à la page Status
type RouteStatus struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	OK        bool      `json:"ok"`
	Status    int       `json:"status"`
	LatencyMs float64   `json:"latency_ms"`
	Uptime    float64   `json:"uptime"` // % de vérifications réussies sur la fenêtre d'historique
	CheckedAt time.Time `json:"checked_at"`
}

// Snapshot : réponse complète de /api/status
type Snapshot struct {
	Overall   string        `json:"overall"` // operational | degraded | down | pending
	CheckedAt time.Time     `json:"checked_at"`
	Interval  string        `json:"interval"`
	Routes    []RouteStatus `json:"routes"`
}

type routeState struct {
	RouteStatus
	history []bool // anneau des derniers résultats (true = OK)
	next    int
	filled  int
}

// Monitor vérifie les routes à intervalle régulier
type Monitor struct {
	handler  http.Handler
	interval time.Duration

	mu        sync.RWMutex
	routes    []*routeState
	lastCheck time.Time
}

// New prépare un moniteur. history = nombre de vérifications gardées par route
// (ex. 1440 vérifications toutes les minutes = 24 h).
func New(handler http.Handler, targets []Target, interval time.Duration, history int) *Monitor {
	if history < 1 {
		history = 1
	}
	m := &Monitor{handler: handler, interval: interval}
	for _, t := range targets {
		m.routes = append(m.routes, &routeState{
			RouteStatus: RouteStatus{Name: t.Name, Path: t.Path},
			history:     make([]bool, history),
		})
	}
	return m
}

// Run lance une vérification immédiate, puis une à chaque intervalle,
// jusqu'à l'arrêt du serveur (ctx annulé).
func (m *Monitor) Run(ctx context.Context) {
	m.CheckAll()
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckAll()
		}
	}
}

// CheckAll vérifie toutes les routes une fois
func (m *Monitor) CheckAll() {
	now := time.Now()
	for _, r := range m.routes {
		status, latency := m.probe(r.Path)
		ok := status >= 200 && status < 400

		m.mu.Lock()
		r.OK = ok
		r.Status = status
		r.LatencyMs = math.Round(float64(latency.Microseconds())/10) / 100 // 2 décimales
		r.CheckedAt = now
		r.history[r.next] = ok
		r.next = (r.next + 1) % len(r.history)
		if r.filled < len(r.history) {
			r.filled++
		}
		r.Uptime = uptime(r.history, r.filled)
		m.mu.Unlock()

		if !ok {
			slog.Warn("monitor : route en échec", "path", r.Path, "status", status)
		}
	}
	m.mu.Lock()
	m.lastCheck = now
	m.mu.Unlock()
}

// probe appelle la route en interne. Un panic dans une page ne doit
// jamais arrêter le moniteur : il est compté comme une erreur 500.
func (m *Monitor) probe(path string) (status int, latency time.Duration) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("User-Agent", "orlandocogo-status-monitor")

	start := time.Now()
	defer func() {
		if err := recover(); err != nil {
			slog.Error("monitor : panic pendant la vérification", "path", path, "error", fmt.Sprint(err))
			status = http.StatusInternalServerError
		}
		latency = time.Since(start)
	}()
	m.handler.ServeHTTP(rec, req)
	return rec.Code, 0 // latency est fixée par le defer
}

func uptime(history []bool, filled int) float64 {
	if filled == 0 {
		return 0
	}
	ok := 0
	for i := 0; i < filled; i++ {
		if history[i] {
			ok++
		}
	}
	return math.Round(float64(ok)/float64(filled)*1000) / 10 // 1 décimale
}

// Snapshot renvoie une copie de l'état actuel (sûre à lire depuis plusieurs goroutines)
func (m *Monitor) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := Snapshot{CheckedAt: m.lastCheck, Interval: m.interval.String(), Overall: "pending"}
	if m.lastCheck.IsZero() {
		return snap
	}
	down := 0
	for _, r := range m.routes {
		snap.Routes = append(snap.Routes, r.RouteStatus)
		if !r.OK {
			down++
		}
	}
	switch {
	case down == 0:
		snap.Overall = "operational"
	case down == len(m.routes):
		snap.Overall = "down"
	default:
		snap.Overall = "degraded"
	}
	return snap
}

// Handler sert le dernier état en JSON (GET /api/status)
func (m *Monitor) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store") // toujours l'état le plus récent
	if err := json.NewEncoder(w).Encode(m.Snapshot()); err != nil {
		slog.Error("monitor : encodage JSON", "error", err)
	}
}