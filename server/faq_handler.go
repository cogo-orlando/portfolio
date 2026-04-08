package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

const faqQuestionsFile = "data/faq_questions.json"

func FAQQuestionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var input struct {
		Question string `json:"question"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Données invalides"})
		return
	}

	input.Question = strings.TrimSpace(input.Question)
	if len(input.Question) < 5 || len(input.Question) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Question invalide"})
		return
	}

	ip := r.RemoteAddr

	if err := saveFAQQuestion(FAQQuestion{
		Question: input.Question,
		Date:     time.Now().Format("2006-01-02 15:04:05"),
		IP:       ip,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur sauvegarde"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func saveFAQQuestion(q FAQQuestion) error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	store := FAQQuestionsStore{}
	if data, err := os.ReadFile(faqQuestionsFile); err == nil {
		json.Unmarshal(data, &store)
	}

	q.ID = store.Total + 1
	store.Total++
	store.Questions = append(store.Questions, q)

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(faqQuestionsFile, data, 0644)
}
