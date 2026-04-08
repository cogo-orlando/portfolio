package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
)

// ── CONFIG ──
var adminPassword = os.Getenv("ADMIN_PASSWORD")

// ── AUTH ──
func checkAdminAuth(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return user == "admin" && pass == adminPassword
}

func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !checkAdminAuth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin"`)
		http.Error(w, "Non autorisé", http.StatusUnauthorized)
		return false
	}
	return true
}

// ── HANDLER ADMIN ──
func AdminHandler(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// ── LOAD DATA ──
	contactStore := MessagesStore{}
	if data, err := os.ReadFile(messagesFile); err == nil {
		json.Unmarshal(data, &contactStore)
	}

	faqStore := FAQQuestionsStore{}
	if data, err := os.ReadFile(faqQuestionsFile); err == nil {
		json.Unmarshal(data, &faqStore)
	}

	// ── HTML ──
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `<h1>Admin Dashboard</h1>`)
	fmt.Fprintf(w, `<p>Messages: %d | FAQ: %d</p>`, contactStore.Total, faqStore.Total)

	// ── CONTACT ──
	fmt.Fprintf(w, `<h2>Messages</h2>`)

	for i := len(contactStore.Messages) - 1; i >= 0; i-- {
		msg := contactStore.Messages[i]

		unread := ""
		if !msg.Read {
			unread = " (NON LU)"
		}

		fmt.Fprintf(w, `
<div style="border:1px solid #ccc;padding:10px;margin:10px;">
<b>#%d %s %s%s</b><br>
%s<br>
<small>%s | %s</small><br>
<a href="/admin/read/%d">Marquer comme lu</a>
</div>
`,
			msg.ID,
			template.HTMLEscapeString(msg.FirstName),
			template.HTMLEscapeString(msg.LastName),
			unread,
			template.HTMLEscapeString(msg.Message),
			template.HTMLEscapeString(msg.Date),
			msg.ID,
		)
	}

	// ── FAQ ──
	fmt.Fprintf(w, `<h2>FAQ</h2>`)

	for i := len(faqStore.Questions) - 1; i >= 0; i-- {
		q := faqStore.Questions[i]

		fmt.Fprintf(w, `
<div style="border:1px solid #ccc;padding:10px;margin:10px;">
<b>#%d</b><br>
%s<br>
<small>%s | %s</small>
</div>
`,
			q.ID,
			template.HTMLEscapeString(q.Question),
			template.HTMLEscapeString(q.Date),
			template.HTMLEscapeString(q.IP),
		)
	}
}

// ── MARK AS READ ──
func AdminMarkReadHandler(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/admin/read/")

	store := MessagesStore{}
	data, err := os.ReadFile(messagesFile)
	if err != nil {
		http.Error(w, "Erreur lecture", 500)
		return
	}

	json.Unmarshal(data, &store)

	for i := range store.Messages {
		if fmt.Sprintf("%d", store.Messages[i].ID) == idStr {
			store.Messages[i].Read = true
		}
	}

	updated, _ := json.MarshalIndent(store, "", "  ")
	os.WriteFile(messagesFile, updated, 0644)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
