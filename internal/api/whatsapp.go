package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type WhatsAppHandler struct {
	Store *store.Store
}

func NewWhatsAppHandler(st *store.Store) *WhatsAppHandler {
	return &WhatsAppHandler{Store: st}
}

// POST /api/whatsapp/send
func (h *WhatsAppHandler) HandleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContactID uint   `json:"contact_id"`
		To        string `json:"to"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// 1. Stub: Call External API (Twilio/Meta)
	// For now, we just log it as "sent"
	msg := models.WhatsAppMessage{
		ContactID: req.ContactID,
		ToNumber:  req.To,
		Body:      req.Body,
		Status:    "sent",
		MessageSID: "WA_MOCK_12345",
		CreatedAt: time.Now(),
	}

	if err := h.Store.DB.Create(&msg).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// POST /api/whatsapp/webhook
func (h *WhatsAppHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 1. Verify Verification Token (Meta Requirement)
	if r.Method == "GET" {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		settings, err := h.Store.GetSettings()
		if err != nil || settings.WhatsAppVerifyToken == "" {
			http.Error(w, "Webhook not configured", http.StatusInternalServerError)
			return
		}

		if mode == "subscribe" && token == settings.WhatsAppVerifyToken {
			w.Write([]byte(challenge))
			return
		}
		http.Error(w, "Verification failed", http.StatusForbidden)
		return
	}

	// 2. Handle Incoming Status/Message
	// ... (Parsing logic omitted for MVP) ...

	w.WriteHeader(http.StatusOK)
}
