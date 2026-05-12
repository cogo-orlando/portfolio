package model

// ══════════════════════════════════════════
//  CONSTANTES PARTAGÉES
// ══════════════════════════════════════════

const MessagesFile = "data/messages.json"

// ══════════════════════════════════════════
//  STRUCTS MESSAGES CONTACT
// ══════════════════════════════════════════

type ContactMessage struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Mail      string `json:"mail"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Date      string `json:"date"`
	IP        string `json:"ip"`
	Read      bool   `json:"read"`
}

type MessagesStore struct {
	Total    int              `json:"total"`
	Messages []ContactMessage `json:"messages"`
}
