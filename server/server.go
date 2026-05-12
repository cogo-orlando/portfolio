package server

import (
	"fmt"
	"net/http"
	"os"
	"portfo/server/handler"
	"portfo/server/middleware"
	"sync"
	"time"
)

var maintenancePages = map[string]bool{
	"/blog": false, "/about": false, "/skills": false,
	"/contact": false, "/cv": false, "/home": false,
	"/project": false, "/faq": false, "/status": false,
	"/demo/annuaire": false, "/demo/netflix": false,
	"/demo/zoo": false, "/demo/power4": false,
	"/demo/groupie": false, "/demo/cisco": false, "/demo/artemis": false,
}

var MaintenanceMode = false

var (
	visitCount int
	visitMu    sync.Mutex
)

var routes = map[string]http.HandlerFunc{
	"/":              handler.IndexHandler,
	"/home":          handler.HomeHandler,
	"/about":         handler.AboutHandler,
	"/skills":        handler.SkillsHandler,
	"/project":       handler.ProjectHandler,
	"/contact":       handler.ContactHandler,
	"/cv":            handler.CvHandler,
	"/status":        handler.StatusHandler,
	"/faq":           handler.FaqHandler,
	"/maintenance":   handler.MaintenanceHandler,
	"/demo/zoo":      handler.DemoZooHandler,
	"/demo/netflix":  handler.DemoNetflixHandler,
	"/demo/groupie":  handler.DemoGroupieHandler,
	"/demo/power4":   handler.DemoPower4Handler,
	"/demo/cisco":    handler.DemoCiscoHandler,
	"/demo/artemis":  handler.DemoArtemisHandler,
	"/demo/annuaire": handler.AnnuaireHandler,
}

func Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", middleware.HealthHandler)
	mux.HandleFunc("/api/contact", handler.ContactAPIHandler)
	mux.HandleFunc("/api/visits", visitsHandler)
	mux.HandleFunc("/admin/messages", handler.AdminHandler)

	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)
	mux.HandleFunc("/", mainHandler)

	h := middleware.Chain(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Serveur lancé sur http://localhost:" + port)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
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
	if MaintenanceMode {
		http.ServeFile(w, r, "web/html/maintenance.html")
		return
	}
	if maintenancePages[r.URL.Path] {
		http.ServeFile(w, r, "web/html/maintenance.html")
		return
	}
	if h, ok := routes[r.URL.Path]; ok {
		h(w, r)
		return
	}
	handler.NotFoundHandler(w, r)
}

func visitsHandler(w http.ResponseWriter, r *http.Request) {
	visitMu.Lock()
	count := visitCount
	visitMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"visits":%d}`, count)
}
