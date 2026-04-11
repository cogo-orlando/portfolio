package server

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

// ══════════════════════════════════════════
//  CONFIGURATION MAINTENANCE
// ══════════════════════════════════════════

// true = page en maintenance
// false = page accessible
var maintenancePages = map[string]bool{
	"/blog":         false,
	"/about":        false,
	"/skills":       false,
	"/contact":      false,
	"/cv":           false,
	"/home":         false,
	"/project":      false,
	"/faq":          false,
	"/status":       false,
	"/demo/netflix": false,
	"/demo/zoo":     false,
	"/demo/power4":  false,
	"/demo/groupie": false,
	"/demo/cisco":   false,
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
//  DÉMARRAGE DU SERVEUR
// ══════════════════════════════════════════

func Start() {
	mux := http.NewServeMux()

	// ── API ──
	mux.HandleFunc("/api/contact", ContactAPIHandler)
	mux.HandleFunc("/api/visits", func(w http.ResponseWriter, r *http.Request) {
		visitMu.Lock()
		count := visitCount
		visitMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"visits":%d}`, count)
	})

	// ── FICHIERS STATIQUES ──
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)

	// ── ROUTES PRINCIPALES ──
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Compteur de visites
		cookie, err := r.Cookie("visited")
		if err != nil || cookie == nil {
			// Nouveau visiteur
			visitMu.Lock()
			visitCount++
			visitMu.Unlock()
			// On pose le cookie pour 24h
			http.SetCookie(w, &http.Cookie{
				Name:   "visited",
				Value:  "1",
				MaxAge: 86400, // 24 heures
				Path:   "/",
			})
		}

		// Site entier en maintenance
		if MaintenanceMode {
			http.ServeFile(w, r, "web/html/maintenance.html")
			return
		}

		// Page spécifique en maintenance
		if maintenancePages[r.URL.Path] {
			http.ServeFile(w, r, "web/html/maintenance.html")
			return
		}

		// Routes
		switch r.URL.Path {

		case "/":
			IndexHandler(w, r)
		case "/home":
			HomeHandler(w, r)
		case "/about":
			AboutHandler(w, r)
		case "/skills":
			SkillsHandler(w, r)
		case "/project":
			ProjectHandler(w, r)
		case "/contact":
			ContactHandler(w, r)
		case "/cv":
			CvHandler(w, r)
		case "/status":
			StatusHandler(w, r)
		case "/faq":
			FaqHandler(w, r)
		case "/maintenance":
			MaintenanceHandler(w, r)
		case "/demo/zoo":
			DemoZooHandler(w, r)
		case "/demo/netflix":
			DemoNetflixHandler(w, r)
		case "/demo/groupie":
			DemoGroupieHandler(w, r)
		case "/demo/power4":
			DemoPower4Handler(w, r)
		case "/demo/cisco":
			DemoCiscoHandler(w, r)

		//Error 404
		default:
			NotFoundHandler(w, r)
		}
	})

	fmt.Println("Serveur lancé sur http://localhost:8080")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		panic(err)
	}
}
