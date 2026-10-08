package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"portfo/server/handler"
	"portfo/server/middleware"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Heure de démarrage, utilisée pour calculer l'uptime dans /health
var startTime = time.Now()

// ══════════════════════════════════════════
//  ROUTES
// ══════════════════════════════════════════

var routes = map[string]http.HandlerFunc{
	"/":                            handler.IndexHandler, // géré à part dans newMux (catch-all + 404)
	"/home":                        handler.HomeHandler,
	"/about":                       handler.AboutHandler,
	"/skills":                      handler.SkillsHandler,
	"/project":                     handler.ProjectHandler,
	"/contact":                     handler.ContactHandler,
	"/cv":                          handler.CvHandler,
	"/status":                      handler.StatusHandler,
	"/faq":                         handler.FaqHandler,
	"/tech":                        handler.TechHandler,
	"/projects/zoo":                handler.ZooHandler,
	"/projects/netflix":            handler.NetflixHandler,
	"/projects/groupie":            handler.GroupieHandler,
	"/projects/power4":             handler.Power4Handler,
	"/projects/cisco":              handler.CiscoHandler,
	"/projects/artemis":            handler.ArtemisHandler,
	"/projects/annuaire":           handler.AnnuaireHandler,
	"/projects/security-dashboard": handler.SecurityDashboardHandler,
	"/projects/forum":              handler.ForumHandler,
	"/projects/snake":              handler.SnakeHandler,
	"/projects/motogp":             handler.MotoGPHandler,
}

// ══════════════════════════════════════════
//  START
// ══════════════════════════════════════════

func Start() {
	env := "production"
	if handler.IsDev() {
		env = "development"
	}

	// Prod : on compile tous les templates tout de suite.
	// Si un HTML est cassé, le serveur refuse de démarrer (visible dans les logs Render)
	// au lieu de servir des pages en erreur.
	count, err := handler.PreloadTemplates()
	if err != nil {
		slog.Error("chargement des templates impossible", "error", err)
		os.Exit(1)
	}
	if count > 0 {
		slog.Info("templates chargés", "count", count)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           middleware.Chain(newMux()),
		ReadHeaderTimeout: 5 * time.Second, // protection Slowloris
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 Mo
	}

	go func() {
		slog.Info("serveur démarré",
			"port", port,
			"env", env,
			"routes", len(routes),
			"url", fmt.Sprintf("http://localhost:%s", port),
		)
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

// newMux construit le routeur. Séparé de Start() pour pouvoir le tester.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	// ── API & fichiers spéciaux ──
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/sitemap.xml", sitemapHandler)
	mux.HandleFunc("/api/visits", visitsHandler)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/img/favicon.ico")
	})

	// ── Fichiers statiques (sans listing de dossiers) ──
	static := noDirListing(http.FileServer(http.Dir("./web")))
	mux.Handle("/css/", static)
	mux.Handle("/js/", static)
	mux.Handle("/img/", static)
	mux.Handle("/robots.txt", static)

	// ── Pages ──
	for path, h := range routes {
		if path == "/" {
			continue
		}
		mux.HandleFunc(path, h)
	}

	// "/" attrape toutes les URL inconnues :
	// exactement "/" → index, tout le reste → page 404 personnalisée
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			handler.NotFoundHandler(w, r)
			return
		}
		handler.IndexHandler(w, r)
	})

	return mux
}

// noDirListing bloque l'affichage du contenu des dossiers (/css/, /js/…).
// Sans ça, http.FileServer liste tous les fichiers d'un dossier :
// pratique pour un attaquant qui fait de la reconnaissance.
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			handler.NotFoundHandler(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════
//  HELPERS
// ══════════════════════════════════════════

// writeJSON encode proprement une réponse JSON (échappement garanti, pas de JSON cassé).
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encodage JSON", "error", err)
	}
}

// allowGet refuse les méthodes autres que GET/HEAD.
func allowGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	return false
}

// ══════════════════════════════════════════
//  HEALTH — métriques Go live
// ══════════════════════════════════════════

type healthResponse struct {
	Status     string  `json:"status"`
	Service    string  `json:"service"`
	Uptime     string  `json:"uptime"`
	Visits     int     `json:"visits"`
	GoVersion  string  `json:"go_version"`
	Goroutines int     `json:"goroutines"`
	MemoryMB   float64 `json:"memory_mb"`
	GCCycles   uint32  `json:"gc_cycles"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		return
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	writeJSON(w, http.StatusOK, healthResponse{
		Status:     "ok",
		Service:    "portfolio",
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		Visits:     0,
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
		MemoryMB:   math.Round(float64(mem.Alloc)/1024/1024*100) / 100,
		GCCycles:   mem.NumGC,
	})
}

// ══════════════════════════════════════════
//  SITEMAP XML
// ══════════════════════════════════════════

var sitemapURLs = []struct {
	path     string
	file     string // template utilisé pour calculer <lastmod>
	priority string
	freq     string
}{
	{"/home", "home.html", "1.0", "weekly"},
	{"/about", "about.html", "0.9", "monthly"},
	{"/skills", "skills.html", "0.9", "monthly"},
	{"/project", "project.html", "0.9", "weekly"},
	{"/contact", "contact.html", "0.8", "monthly"},
	{"/cv", "cv.html", "0.8", "monthly"},
	{"/faq", "faq.html", "0.7", "monthly"},
	{"/tech", "tech.html", "0.8", "monthly"},
	{"/projects/security-dashboard", "projects/security-dashboard.html", "0.8", "weekly"},
	{"/projects/forum", "projects/forum.html", "0.6", "monthly"},
	{"/projects/motogp", "projects/motogp.html", "0.6", "monthly"},
	{"/projects/netflix", "projects/netflix.html", "0.6", "monthly"},
	{"/projects/groupie", "projects/groupie.html", "0.6", "monthly"},
	{"/projects/power4", "projects/power4.html", "0.6", "monthly"},
	{"/projects/zoo", "projects/zoo.html", "0.6", "monthly"},
	{"/projects/cisco", "projects/cisco.html", "0.6", "monthly"},
	{"/projects/artemis", "projects/artemis.html", "0.6", "monthly"},
	{"/projects/annuaire", "projects/annuaire.html", "0.6", "monthly"},
	{"/projects/snake", "projects/snake.html", "0.6", "monthly"},
}

// lastMod renvoie la vraie date de dernière modif du template
// (Google se fie à cette date : la mettre à "aujourd'hui" à chaque fois la rend inutile).
func lastMod(file string) string {
	info, err := os.Stat(filepath.Join("web", "html", filepath.FromSlash(file)))
	if err != nil {
		return time.Now().Format("2006-01-02")
	}
	return info.ModTime().Format("2006-01-02")
}

func sitemapHandler(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		return
	}

	const base = "https://orlandocogo.com"

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range sitemapURLs {
		fmt.Fprintf(&b, "  <url>\n")
		fmt.Fprintf(&b, "    <loc>%s%s</loc>\n", base, u.path)
		fmt.Fprintf(&b, "    <lastmod>%s</lastmod>\n", lastMod(u.file))
		fmt.Fprintf(&b, "    <changefreq>%s</changefreq>\n", u.freq)
		fmt.Fprintf(&b, "    <priority>%s</priority>\n", u.priority)
		fmt.Fprintf(&b, "  </url>\n")
	}
	b.WriteString(`</urlset>`)

	if _, err := w.Write([]byte(b.String())); err != nil {
		slog.Debug("écriture sitemap interrompue", "error", err)
	}
}

// ══════════════════════════════════════════
//  API VISITS
// ══════════════════════════════════════════

type visitsResponse struct {
	Visits int `json:"visits"`
}

func visitsHandler(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, visitsResponse{Visits: 0})
}