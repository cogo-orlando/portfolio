package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	messagesFile = "data/messages.json"
)

// ── RATE LIMITING ──
type RateLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

var rateLimiter = &RateLimiter{records: make(map[string][]time.Time)}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := time.Hour

	// Nettoie les vieilles entrées
	var recent []time.Time
	for _, t := range rl.records[ip] {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}

	// Max 3 messages par heure par IP
	if len(recent) >= 100 {
		return false
	}

	rl.records[ip] = append(recent, now)
	return true
}

// ── HANDLER CONTACT ──
func ContactAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse le body JSON
	var input struct {
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
		Mail      string `json:"mail"`
		Subject   string `json:"subject"`
		Message   string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Données invalides"})
		return
	}

	// ── VALIDATION CÔTÉ SERVEUR ──
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Mail = strings.TrimSpace(input.Mail)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Message = strings.TrimSpace(input.Message)

	if input.FirstName == "" {
		jsonError(w, "Le nom est requis", http.StatusBadRequest)
		return
	}
	if len(input.FirstName) > 100 {
		jsonError(w, "Nom trop long", http.StatusBadRequest)
		return
	}

	if input.LastName == "" {
		jsonError(w, "Le nom est requis", http.StatusBadRequest)
		return
	}
	if len(input.LastName) > 100 {
		jsonError(w, "Nom trop long", http.StatusBadRequest)
		return
	}

	if input.Mail == "" {
		jsonError(w, "Un mail est requis", http.StatusBadRequest)
		return
	}
	if len(input.Mail) > 1000 {
		jsonError(w, "Le mail est trop long", http.StatusBadRequest)
		return
	}

	validSubjects := map[string]bool{"stage": true, "projet": true, "question": true, "autre": true}
	if !validSubjects[input.Subject] {
		jsonError(w, "Sujet invalide", http.StatusBadRequest)
		return
	}

	if len(input.Message) < 10 {
		jsonError(w, "Message trop court", http.StatusBadRequest)
		return
	}
	if len(input.Message) > 1000 {
		jsonError(w, "Message trop long", http.StatusBadRequest)
		return
	}

	// ── SAUVEGARDE EN JSON ──
	msg := ContactMessage{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Mail:      input.Mail,
		Subject:   input.Subject,
		Message:   input.Message,
		Date:      time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := saveMessage(msg); err != nil {
		fmt.Println("[CONTACT] Erreur sauvegarde:", err)
		jsonError(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ── SAUVEGARDE JSON ──
func saveMessage(msg ContactMessage) error {
	// Crée le dossier data/ si nécessaire
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	// Lit le fichier existant ou crée un nouveau store
	store := MessagesStore{}
	if data, err := os.ReadFile(messagesFile); err == nil {
		json.Unmarshal(data, &store)
	}

	// Écrit le fichier
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(messagesFile, data, 0644)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
