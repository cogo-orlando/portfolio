package server

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// ══════════════════════════════════════════
//  CONFIGURATION MAINTENANCE
// ══════════════════════════════════════════

var maintenancePages = map[string]bool{
	"/blog":          false,
	"/about":         false,
	"/skills":        false,
	"/contact":       false,
	"/cv":            false,
	"/home":          false,
	"/project":       false,
	"/faq":           false,
	"/status":        false,
	"/demo/annuaire": false,
	"/demo/netflix":  false,
	"/demo/zoo":      false,
	"/demo/power4":   false,
	"/demo/groupie":  false,
	"/demo/cisco":    false,
	"/demo/artemis":  false,
}

var MaintenanceMode = false

// ══════════════════════════════════════════
//  COMPTEUR DE VISITES
// ══════════════════════════════════════════

var (
	visitCount int
	visitMu    sync.Mutex
)

// ══════════════════════════════════════════
//  TABLE DE ROUTAGE
// ══════════════════════════════════════════

var routes = map[string]http.HandlerFunc{
	"/":              IndexHandler,
	"/home":          HomeHandler,
	"/about":         AboutHandler,
	"/skills":        SkillsHandler,
	"/project":       ProjectHandler,
	"/contact":       ContactHandler,
	"/cv":            CvHandler,
	"/status":        StatusHandler,
	"/faq":           FaqHandler,
	"/maintenance":   MaintenanceHandler,
	"/demo/zoo":      DemoZooHandler,
	"/demo/netflix":  DemoNetflixHandler,
	"/demo/groupie":  DemoGroupieHandler,
	"/demo/power4":   DemoPower4Handler,
	"/demo/cisco":    DemoCiscoHandler,
	"/demo/artemis":  DemoArtemisHandler,
	"/demo/annuaire": AnnuaireHandler,
}

// ══════════════════════════════════════════
//  DÉMARRAGE DU SERVEUR
// ══════════════════════════════════════════

func Start() {
	mux := http.NewServeMux()

	// ── HEALTH CHECK ──
	mux.HandleFunc("/health", healthHandler)

	// ── API ──
	mux.HandleFunc("/api/contact", ContactAPIHandler)
	mux.HandleFunc("/api/visits", visitsHandler)

	// ── ADMIN ──
	mux.HandleFunc("/admin/messages", AdminHandler)

	// ── FICHIERS STATIQUES ──
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)

	// ── ROUTES PRINCIPALES ──
	mux.HandleFunc("/", mainHandler)

	// ── APPLIQUE LE MIDDLEWARE ──
	handler := Chain(mux)

	fmt.Println("Serveur lancé sur http://localhost:8080")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}

// ── HANDLER PRINCIPAL ──
func mainHandler(w http.ResponseWriter, r *http.Request) {
	// Compteur — 1 visite par 24h via cookie
	if _, err := r.Cookie("visited"); err != nil {
		visitMu.Lock()
		visitCount++
		visitMu.Unlock()
		http.SetCookie(w, &http.Cookie{
			Name:     "visited",
			Value:    "1",
			MaxAge:   86400,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	}

	// Maintenance globale
	if MaintenanceMode {
		http.ServeFile(w, r, "web/html/maintenance.html")
		return
	}

	// Maintenance par page
	if maintenancePages[r.URL.Path] {
		http.ServeFile(w, r, "web/html/maintenance.html")
		return
	}

	// Lookup dans la table de routage
	if handler, ok := routes[r.URL.Path]; ok {
		handler(w, r)
		return
	}

	NotFoundHandler(w, r)
}

// ── API VISITS ──
func visitsHandler(w http.ResponseWriter, r *http.Request) {
	visitMu.Lock()
	count := visitCount
	visitMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"visits":%d}`, count)
}
