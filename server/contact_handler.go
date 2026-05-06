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

const messagesFile = "data/messages.json"

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

	// Garde uniquement les entrées dans la fenêtre d'une heure
	var recent []time.Time
	for _, t := range rl.records[ip] {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}

	// Max 3 messages par heure par IP
	if len(recent) >= 3 {
		return false
	}

	rl.records[ip] = append(recent, now)
	return true
}

// ── VALIDATION ──
// validateField vérifie qu'un champ est non vide et dans la limite de taille
func validateField(w http.ResponseWriter, value, name string, maxLen int) bool {
	if strings.TrimSpace(value) == "" {
		jsonError(w, name+" est requis", http.StatusBadRequest)
		return false
	}
	if len(value) > maxLen {
		jsonError(w, name+" est trop long", http.StatusBadRequest)
		return false
	}
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
		jsonError(w, "Données invalides", http.StatusBadRequest)
		return
	}

	// Trim tous les champs
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Mail = strings.TrimSpace(input.Mail)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Message = strings.TrimSpace(input.Message)

	// Validation factorisée
	if !validateField(w, input.FirstName, "Le prénom", 100) {
		return
	}
	if !validateField(w, input.LastName, "Le nom", 100) {
		return
	}
	if !validateField(w, input.Mail, "L'email", 254) {
		return
	}

	validSubjects := map[string]bool{"stage": true, "projet": true, "question": true, "autre": true}
	if !validSubjects[input.Subject] {
		jsonError(w, "Sujet invalide", http.StatusBadRequest)
		return
	}

	if len(input.Message) < 10 {
		jsonError(w, "Message trop court (min 10 caractères)", http.StatusBadRequest)
		return
	}
	if len(input.Message) > 1000 {
		jsonError(w, "Message trop long (max 1000 caractères)", http.StatusBadRequest)
		return
	}

	// Rate limiting par IP
	ip := r.RemoteAddr
	if !rateLimiter.Allow(ip) {
		jsonError(w, "Trop de messages envoyés, réessaie dans une heure", http.StatusTooManyRequests)
		return
	}

	// Sauvegarde
	msg := ContactMessage{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Mail:      input.Mail,
		Subject:   input.Subject,
		Message:   input.Message,
		Date:      time.Now().Format("2006-01-02 15:04:05"),
		IP:        ip,
	}

	if err := saveMessage(msg); err != nil {
		fmt.Println("[CONTACT] Erreur sauvegarde:", err)
		jsonError(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── SAUVEGARDE JSON ──
func saveMessage(msg ContactMessage) error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	// Charge le store existant
	store := MessagesStore{}
	if data, err := os.ReadFile(messagesFile); err == nil {
		json.Unmarshal(data, &store)
	}

	// Assigne un ID et ajoute le message
	msg.ID = store.Total + 1
	store.Total++
	store.Messages = append(store.Messages, msg)

	// Écrit le fichier
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(messagesFile, data, 0644)
}

// ── HELPER ──
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
