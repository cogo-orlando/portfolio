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

// PAGE BIENVENUE
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html")
}

// PAGE D'ACCUEIL
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home.html")
}

// PAGE A PROPOS DE MOI
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html")
}

// PAGE MES COMPETENCES
func SkillsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "skills.html")
}

// PAGE MES PROJETS
func ProjectHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "project.html")
}

// PAGE CONTACT
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "contact.html")
}

// PAGE CV
func CvHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "cv.html")
}

// PAGE STATUS DU SITE
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "status.html")
}

// PAGE FAQ
func FaqHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "faq.html")
}

// PAGE MAINTENANCE
func MaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "maintenance.html")
}

// PAGE ERROR 404
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404.html")
}
