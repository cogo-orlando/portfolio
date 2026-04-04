package server

import (
	"fmt"
	"net/http"
)

// ── MAINTENANCE MODE ──
// true  = site en maintenance
// false = site normal
var MaintenanceMode = false

func Start() {
	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/api/contact", ContactAPIHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		case "/blog":
			BlogHandler(w, r)
		case "/uses":
			UsesHandler(w, r)
		case "/maintenance":
			MaintenanceHandler(w, r)
		default:
			NotFoundHandler(w, r)
		}
	})

	// Fichiers statiques
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)

	// Middleware maintenance — entoure tout le mux
	var handler http.Handler = mux
	if MaintenanceMode {
		handler = maintenanceMiddleware(mux)
	}

	fmt.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}

// maintenanceMiddleware redirige toutes les requêtes vers la page maintenance
// sauf les fichiers statiques (css, js, img) pour que la page soit belle
func maintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Laisse passer les fichiers statiques
		if len(r.URL.Path) > 4 {
			prefix := r.URL.Path[:5]
			if prefix == "/css/" || prefix == "/img/" {
				next.ServeHTTP(w, r)
				return
			}
		}
		if len(r.URL.Path) > 3 && r.URL.Path[:4] == "/js/" {
			next.ServeHTTP(w, r)
			return
		}

		// Tout le reste → page maintenance
		http.ServeFile(w, r, "web/html/maintenance.html")
	})
}
