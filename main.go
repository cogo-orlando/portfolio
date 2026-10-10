package main

import (
	"net/http"
	"os"
	"time"

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
func healthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
