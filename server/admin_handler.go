package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
)

// ── MOT DE PASSE ADMIN — depuis variable d'environnement ──
var adminPassword = os.Getenv("ADMIN_PASSWORD")

// ── VÉRIFICATION MOT DE PASSE ──
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

// ── HANDLER PRINCIPAL ADMIN ──
func AdminHandler(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// Charge les messages de contact
	store := MessagesStore{}
	if data, err := os.ReadFile(messagesFile); err == nil {
		json.Unmarshal(data, &store)
	}

	subjectLabels := map[string]string{
		"stage":    "Opportunité de stage",
		"projet":   "Proposition de projet",
		"question": "Question",
		"autre":    "Autre",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Admin — Messages</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#080b0f;--accent:#00f5a0;--text:#e8f0f8;--muted2:#5a7080;--surface:rgba(255,255,255,0.03);--border2:rgba(255,255,255,0.07)}
body{background:var(--bg);color:var(--text);font-family:'DM Mono',monospace;padding:2rem;min-height:100vh}
h1{font-family:sans-serif;font-size:1.5rem;font-weight:800;color:var(--accent);margin-bottom:0.5rem}
h2{font-family:sans-serif;font-size:1.1rem;font-weight:700;color:var(--text);margin:2rem 0 1rem;border-bottom:1px solid var(--border2);padding-bottom:0.5rem}
.meta{font-size:11px;color:var(--muted2);margin-bottom:2rem}
.card{background:var(--surface);border:1px solid var(--border2);padding:1.2rem 1.4rem;margin-bottom:0.75rem}
.card:hover{border-color:rgba(0,245,160,0.2)}
.card-header{display:flex;justify-content:space-between;align-items:center;margin-bottom:0.75rem;flex-wrap:wrap;gap:8px}
.card-name{font-size:14px;font-weight:700;color:var(--text)}
.card-date{font-size:10px;color:var(--muted2)}
.card-subject{font-size:10px;color:var(--accent);border:1px solid rgba(0,245,160,0.3);padding:2px 8px}
.card-msg{font-size:13px;color:#c8d4e0;line-height:1.7;margin-top:0.5rem}
.card-ip{font-size:10px;color:var(--muted2);margin-top:0.5rem}
.card-mail{font-size:11px;color:var(--accent);margin-top:0.25rem}
.card-id{font-size:10px;color:var(--muted2)}
.unread{border-left:2px solid var(--accent)}
.empty{font-size:13px;color:var(--muted2);padding:1rem 0}
.stat{display:inline-flex;flex-direction:column;gap:2px;margin-right:2rem}
.stat-val{font-size:1.5rem;font-weight:800;color:var(--accent)}
.stat-label{font-size:10px;color:var(--muted2);letter-spacing:0.1em}
.back{display:inline-block;margin-top:2rem;font-size:12px;color:var(--accent);text-decoration:none}
.tag{font-size:10px;color:var(--accent);letter-spacing:0.1em}
</style>
</head>
<body>
<p class="tag">~/admin $</p>
<h1>Dashboard Admin</h1>
<p class="meta">Portfolio Orlando Cogo · Accès restreint</p>
<div style="margin-bottom:2rem">
  <div class="stat"><span class="stat-val">%d</span><span class="stat-label">Messages reçus</span></div>
</div>
<h2>Messages de contact</h2>`, store.Total)

	if len(store.Messages) == 0 {
		fmt.Fprintf(w, `<p class="empty">Aucun message pour le moment.</p>`)
	} else {
		for i := len(store.Messages) - 1; i >= 0; i-- {
			msg := store.Messages[i]
			subjectLabel := subjectLabels[msg.Subject]
			if subjectLabel == "" {
				subjectLabel = msg.Subject
			}
			unreadClass := ""
			if !msg.Read {
				unreadClass = " unread"
			}
			fmt.Fprintf(w, `
<div class="card%s">
  <div class="card-header">
    <span class="card-id">#%d</span>
    <span class="card-name">%s %s</span>
    <span class="card-subject">%s</span>
    <span class="card-date">%s</span>
  </div>
  <div class="card-mail">✉ %s</div>
  <div class="card-msg">%s</div>
  <div class="card-ip">IP : %s</div>
</div>`,
				unreadClass,
				msg.ID,
				template.HTMLEscapeString(msg.FirstName),
				template.HTMLEscapeString(msg.LastName),
				template.HTMLEscapeString(subjectLabel),
				template.HTMLEscapeString(msg.Date),
				template.HTMLEscapeString(msg.Mail),
				template.HTMLEscapeString(msg.Message),
				template.HTMLEscapeString(msg.IP),
			)
		}
	}

	fmt.Fprintf(w, `<a href="/home" class="back">← Retour au site</a></body></html>`)
}

// ── MARQUER UN MESSAGE COMME LU ──
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
	http.Redirect(w, r, "/admin/messages", http.StatusSeeOther)
}
