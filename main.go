package main

import (
	"net/http"
	"os"
	"time"
	"fmt"
	"strconv"

	"portfo/server"
	"portfo/server/db"
)

func main() {
	// Healthcheck Docker : "portfolio healthcheck" interroge /health puis s'arrête.
	// (l'image scratch n'a ni shell ni wget)
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	// Initialise la connexion Supabase
	// Si DATABASE_URL n'est pas définie, ignore silencieusement
	db.Init()
	defer db.Close()

	server.Start()
}

// healthcheck renvoie 0 si /health répond 200, 1 sinon
// healthcheck renvoie 0 si /health répond 200, 1 sinon
func healthcheck() int {
	// Le port vient de l'environnement : on vérifie que c'est bien un numéro de port
	// (sinon "8080@autre-site.com" pourrait détourner la requête ailleurs).
	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return 1
		}
		port = p
	}

	client := http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	resp, err := client.Get(url) // #nosec G704 -- hôte fixe (127.0.0.1), port validé (entier 1-65535)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
