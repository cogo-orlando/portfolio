package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"portfo/server/handler"
	"portfo/server/middleware"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ══════════════════════════════════════════
//  MAINTENANCE
// ══════════════════════════════════════════

var maintenancePages = map[string]bool{
	"/blog": false, "/about": false, "/skills": false,
	"/contact": false, "/cv": false, "/home": false,
	"/project": false, "/faq": false, "/status": false,
	"/tech":          false,
	"/demo/annuaire": false, "/demo/netflix": false,
	"/demo/zoo": false, "/demo/power4": false,
	"/demo/groupie": false, "/demo/cisco": false, "/demo/artemis": false,
}

var MaintenanceMode = false

// ══════════════════════════════════════════
//  COMPTEUR DE VISITES
// ══════════════════════════════════════════

var (
	visitCount  int
	visitMu     sync.Mutex
	serverStart = time.Now()
)

// ══════════════════════════════════════════
//  ROUTES
// ══════════════════════════════════════════

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
	"/tech":          handler.TechHandler,
	"/maintenance":   handler.MaintenanceHandler,
	"/demo/zoo":      handler.DemoZooHandler,
	"/demo/netflix":  handler.DemoNetflixHandler,
	"/demo/groupie":  handler.DemoGroupieHandler,
	"/demo/power4":   handler.DemoPower4Handler,
	"/demo/cisco":    handler.DemoCiscoHandler,
	"/demo/artemis":  handler.DemoArtemisHandler,
	"/demo/annuaire": handler.AnnuaireHandler,
}

// ══════════════════════════════════════════
//  START
// ══════════════════════════════════════════

func Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/sitemap.xml", sitemapHandler)
	mux.HandleFunc("/api/visits", visitsHandler)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/img/favicon.ico")
	})

	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/css/", fs)
	mux.Handle("/js/", fs)
	mux.Handle("/img/", fs)
	mux.Handle("/robots.txt", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/", mainHandler)

	h := middleware.Chain(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("serveur démarré", "port", port, "url", fmt.Sprintf("http://localhost:%s", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("erreur fatale serveur", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("arrêt gracieux en cours", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("arrêt forcé", "error", err)
	} else {
		slog.Info("arrêt propre — toutes les requêtes terminées")
	}
}

// ══════════════════════════════════════════
//  MAIN HANDLER
// ══════════════════════════════════════════

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

// ══════════════════════════════════════════
//  HEALTH — enrichi avec métriques Go
// ══════════════════════════════════════════

func healthHandler(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	visitMu.Lock()
	visits := visitCount
	visitMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{`+
		`"status":"ok",`+
		`"service":"portfolio",`+
		`"uptime":"%s",`+
		`"visits":%d,`+
		`"go_version":"%s",`+
		`"goroutines":%d,`+
		`"memory_mb":%.2f,`+
		`"gc_cycles":%d`+
		`}`,
		time.Since(serverStart).Round(time.Second).String(),
		visits,
		runtime.Version(),
		runtime.NumGoroutine(),
		float64(mem.Alloc)/1024/1024,
		mem.NumGC,
	)
}

// ══════════════════════════════════════════
//  SITEMAP XML
// ══════════════════════════════════════════

var sitemapURLs = []struct {
	path     string
	priority string
	freq     string
}{
	{"/home", "1.0", "weekly"},
	{"/about", "0.9", "monthly"},
	{"/skills", "0.9", "monthly"},
	{"/project", "0.9", "weekly"},
	{"/contact", "0.8", "monthly"},
	{"/cv", "0.8", "monthly"},
	{"/faq", "0.7", "monthly"},
	{"/tech", "0.8", "monthly"},
}

func sitemapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	now := time.Now().Format("2006-01-02")
	base := "https://orlandocogo.com"

	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")

	for _, u := range sitemapURLs {
		fmt.Fprintf(w, "  <url>\n")
		fmt.Fprintf(w, "    <loc>%s%s</loc>\n", base, u.path)
		fmt.Fprintf(w, "    <lastmod>%s</lastmod>\n", now)
		fmt.Fprintf(w, "    <changefreq>%s</changefreq>\n", u.freq)
		fmt.Fprintf(w, "    <priority>%s</priority>\n", u.priority)
		fmt.Fprintf(w, "  </url>\n")
	}

	fmt.Fprint(w, `</urlset>`)
}

// ══════════════════════════════════════════
//  API VISITS
// ══════════════════════════════════════════

func visitsHandler(w http.ResponseWriter, r *http.Request) {
	visitMu.Lock()
	count := visitCount
	visitMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"visits":%d}`, count)
}
