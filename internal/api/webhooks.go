package api

import (
	"encoding/json"
	"net/http"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// GET /api/webhooks/settings
func (s *Server) handleGetWebhookSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"webhook_url":      "",
			"webhook_enabled":  false,
			"bounce_alert_pct": 5.0,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"webhook_url":      settings.WebhookURL,
		"webhook_enabled":  settings.WebhookEnabled,
		"bounce_alert_pct": settings.BounceAlertPct,
	})
}

// POST /api/webhooks/settings
func (s *Server) handleSetWebhookSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WebhookURL     string  `json:"webhook_url"`
		WebhookEnabled bool    `json:"webhook_enabled"`
		BounceAlertPct float64 `json:"bounce_alert_pct"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	settings, err := s.Store.GetSettings()
	if err != nil {
		// Create new settings
		settings = &models.AppSettings{}
	}

	settings.WebhookURL = req.WebhookURL
	settings.WebhookEnabled = req.WebhookEnabled
	settings.BounceAlertPct = req.BounceAlertPct

	if settings.BounceAlertPct <= 0 {
		settings.BounceAlertPct = 5.0 // Default 5%
	}

	if err := s.Store.UpsertSettings(settings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// POST /api/webhooks/test
func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WebhookURL string `json:"webhook_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.WebhookURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "webhook_url required"})
		return
	}

	ws := core.NewWebhookService(s.Store)
	if err := ws.SendTestWebhook(req.WebhookURL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// GET /api/webhooks/logs
func (s *Server) handleGetWebhookLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.Store.ListWebhookLogs(50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get logs"})
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// POST /api/webhooks/check-bounces
func (s *Server) handleCheckBounces(w http.ResponseWriter, r *http.Request) {
	ws := core.NewWebhookService(s.Store)
	if err := ws.CheckBounceRates(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check bounces"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "checked"})
}
