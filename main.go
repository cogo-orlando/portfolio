package main

import (
	"portfo/server"
	"portfo/server/db"
)

func main() {
	// Initialise la connexion Supabase
	// Si DATABASE_URL n'est pas définie, ignore silencieusement
	db.Init()
	defer db.Close()

	server.Start()
}
