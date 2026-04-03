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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html")
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home.html")
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html")
}

func SkillsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "skills.html")
}

func ProjectHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "project.html")
}

func ContactHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "contact.html")
}

func CvHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "cv.html")
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "status.html")
}

func FaqHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "faq.html")
}

func BlogHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "blog.html")
}

func UsesHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "uses.html")
}

func MaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "maintenance.html")
}

// Error 404
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404.html")
}
