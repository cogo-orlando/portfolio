package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"portfo/server/model"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ══════════════════════════════════════════
//  RATE LIMITER CONTACT (3 msg/heure/IP)
// ══════════════════════════════════════════

type contactLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

var contactRateLimiter = &contactLimiter{records: make(map[string][]time.Time)}

func (cl *contactLimiter) allow(ip string) bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	now := time.Now()
	var recent []time.Time
	for _, t := range cl.records[ip] {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 3 {
		return false
	}
	cl.records[ip] = append(recent, now)
	return true
}

func init() {
	go func() {
		for range time.Tick(time.Hour) {
			contactRateLimiter.mu.Lock()
			now := time.Now()
			for ip, times := range contactRateLimiter.records {
				var recent []time.Time
				for _, t := range times {
					if now.Sub(t) < time.Hour {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(contactRateLimiter.records, ip)
				} else {
					contactRateLimiter.records[ip] = recent
				}
			}
			contactRateLimiter.mu.Unlock()
		}
	}()
}

// ══════════════════════════════════════════
//  VALIDATION
// ══════════════════════════════════════════

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var validSubjects = map[string]bool{
	"stage": true, "projet": true, "question": true, "autre": true,
}

// sanitize supprime les caractères de contrôle dangereux
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 || r == '\n' || r == '\t' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

type contactInput struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Mail      string `json:"mail"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}

func validateContact(input *contactInput) string {
	input.FirstName = sanitize(input.FirstName)
	input.LastName = sanitize(input.LastName)
	input.Mail = sanitize(input.Mail)
	input.Subject = sanitize(input.Subject)
	input.Message = sanitize(input.Message)

	if utf8.RuneCountInString(input.FirstName) == 0 {
		return "Le prénom est requis"
	}
	if utf8.RuneCountInString(input.FirstName) > 100 {
		return "Le prénom est trop long"
	}
	if utf8.RuneCountInString(input.LastName) == 0 {
		return "Le nom est requis"
	}
	if utf8.RuneCountInString(input.LastName) > 100 {
		return "Le nom est trop long"
	}
	if input.Mail == "" {
		return "L'email est requis"
	}
	if len(input.Mail) > 254 || !emailRegex.MatchString(input.Mail) {
		return "Email invalide"
	}
	if !validSubjects[input.Subject] {
		return "Sujet invalide"
	}
	msgLen := utf8.RuneCountInString(input.Message)
	if msgLen < 10 {
		return "Message trop court (min 10 caractères)"
	}
	if msgLen > 1000 {
		return "Message trop long (max 1000 caractères)"
	}
	return ""
}

// ══════════════════════════════════════════
//  IP RÉELLE VIA CLOUDFLARE
// ══════════════════════════════════════════

func getRealIP(r *http.Request) string {
	// Cloudflare injecte CF-Connecting-IP avec l'IP réelle du visiteur
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	return r.RemoteAddr
}

// ══════════════════════════════════════════
//  HANDLER CONTACT
// ══════════════════════════════════════════

func ContactAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		jsonError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Limite le body à 10KB — protection contre les gros payloads
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024)

	var input contactInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Données invalides", http.StatusBadRequest)
		return
	}

	if errMsg := validateContact(&input); errMsg != "" {
		jsonError(w, errMsg, http.StatusBadRequest)
		return
	}

	ip := getRealIP(r)
	if !contactRateLimiter.allow(ip) {
		log.Printf("[SECURITY] Rate limit contact — IP: %s", ip)
		jsonError(w, "Trop de messages envoyés, réessaie dans une heure", http.StatusTooManyRequests)
		return
	}

	msg := model.ContactMessage{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Mail:      input.Mail,
		Subject:   input.Subject,
		Message:   input.Message,
		Date:      time.Now().Format("02/01/2006 15:04"),
		IP:        ip,
	}

	if err := saveMessage(msg); err != nil {
		log.Printf("[ERROR] Sauvegarde contact: %v", err)
		jsonError(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	log.Printf("[CONTACT] Message de %s %s — IP: %s", input.FirstName, input.LastName, ip)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ══════════════════════════════════════════
//  SAUVEGARDE JSON (mutex pour éviter les race conditions)
// ══════════════════════════════════════════

var saveMu sync.Mutex

func saveMessage(msg model.ContactMessage) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	store := model.MessagesStore{}
	if data, err := os.ReadFile(model.MessagesFile); err == nil {
		json.Unmarshal(data, &store)
	}

	msg.ID = store.Total + 1
	store.Total++
	store.Messages = append(store.Messages, msg)

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(model.MessagesFile, data, 0644)
}

// ══════════════════════════════════════════
//  HELPER
// ══════════════════════════════════════════

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
