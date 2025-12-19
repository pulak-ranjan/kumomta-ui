package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type ContactHandler struct {
	Store *store.Store
}

func NewContactHandler(st *store.Store) *ContactHandler {
	return &ContactHandler{Store: st}
}

// POST /api/contacts/verify
func (h *ContactHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Fetch Hostname
	hostname := "kumomta.local"
	var proxyURL string
	if s, err := h.Store.GetSettings(); err == nil {
		if s.MainHostname != "" { hostname = s.MainHostname }
		proxyURL = s.ProxyURL
	}

	opts := core.VerifierOptions{
		HeloHost: hostname,
		ProxyURL: proxyURL,
	}

	result := core.VerifyEmail(req.Email, opts)
	writeJSON(w, http.StatusOK, result)
}

// POST /api/lists/{id}/clean
func (h *ContactHandler) HandleCleanList(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	// Fetch contacts
	var contacts []models.Contact
	if err := h.Store.DB.Where("list_id = ?", id).Find(&contacts).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	// Fetch Hostname
	hostname := "kumomta.local"
	var proxyURL string
	if s, err := h.Store.GetSettings(); err == nil {
		if s.MainHostname != "" { hostname = s.MainHostname }
		proxyURL = s.ProxyURL
	}

	opts := core.VerifierOptions{
		HeloHost: hostname,
		ProxyURL: proxyURL,
	}

	// Run cleaning in background (simple approach)
	go func() {
		for _, c := range contacts {
			res := core.VerifyEmail(c.Email, opts)

			c.IsValid = res.IsValid
			c.RiskScore = res.RiskScore
			c.VerifyLog = res.Log

			h.Store.DB.Save(&c)

			// Throttle slightly
			time.Sleep(100 * time.Millisecond)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "cleaning_started",
		"count": strconv.Itoa(len(contacts)),
	})
}
