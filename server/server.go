package server

import (
	"fmt"
	"net/http"
	"os"
)

// ══════════════════════════════════════════
//  CONFIGURATION MAINTENANCE
// ══════════════════════════════════════════

// true = page en maintenance
// false = page accessible
var maintenancePages = map[string]bool{
	"/blog":    false,
	"/about":   false,
	"/skills":  false,
	"/contact": false,
	"/cv":      false,
	"/home":    false,
	"/project": false,
	"/faq":     false,
	"/status":  false,
}

var MaintenanceMode = false

// ══════════════════════════════════════════
//  DÉMARRAGE DU SERVEUR
// ══════════════════════════════════════════

func Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/messages", AdminHandler)

	// ── API ──
	mux.HandleFunc("/api/contact", ContactAPIHandler)
	mux.HandleFunc("/api/faq-question", FAQQuestionHandler)

	// ── FICHIERS STATIQUES ──
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)

	// ── ROUTES ──
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Si le site entier est en maintenance → tout le monde voit maintenance.html
		if MaintenanceMode {
			http.ServeFile(w, r, "web/html/maintenance.html")
			return
		}

		// Si la page demandée est en maintenance → affiche maintenance.html
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
		default:
			NotFoundHandler(w, r)
		}
	})

	fmt.Println("Serveur lancé sur http://localhost:8080")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Serveur lancé sur :" + port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		panic(err)
	}
}
