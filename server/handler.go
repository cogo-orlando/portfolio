package server

import (
	"html/template"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, file string) {
	tmpl, err := template.ParseFiles("web/html/" + file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "welcome.html")
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home.html")
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html")
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404.html")
}
