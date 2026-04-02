package server

import (
	"fmt"
	"net/http"
)

func Start() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {

		case "/":
			WelcomeHandler(w, r)
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

		//Error 404
		default:
			NotFoundHandler(w, r)
		}
	})

	// File Static
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/css/", fs)
	http.Handle("/js/", fs)
	http.Handle("/img/", fs)

	// LocalHost 8080
	fmt.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
