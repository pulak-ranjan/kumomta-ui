package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// DTO for the frontend table
type WarmupDTO struct {
	SenderID    uint      `json:"sender_id"`
	Email       string    `json:"email"`
	Domain      string    `json:"domain"`
	Enabled     bool      `json:"enabled"`
	Plan        string    `json:"plan"`
	Day         int       `json:"day"`
	CurrentRate string    `json:"current_rate"`
	LastUpdate  time.Time `json:"last_update"`
}

// GET /api/warmup
// Returns a list of all senders + their current warmup status/rate
func (s *Server) handleGetWarmupList(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list domains"})
		return
	}

	var list []WarmupDTO

	for _, d := range domains {
		for _, snd := range d.Senders {
			// Calculate what the current rate is right now
			rate := core.GetSenderRate(snd)
			
			// Friendly display text
			if rate == "" && snd.WarmupEnabled {
				rate = "Complete"
			} else if rate == "" {
				rate = "Unlimited"
			}

			list = append(list, WarmupDTO{
				SenderID:    snd.ID,
				Email:       snd.Email,
				Domain:      d.Name,
				Enabled:     snd.WarmupEnabled,
				Plan:        snd.WarmupPlan,
				Day:         snd.WarmupDay,
				CurrentRate: rate,
				LastUpdate:  snd.WarmupLastUpdate,
			})
		}
	}

	writeJSON(w, http.StatusOK, list)
}

// POST /api/warmup/{id}
// Toggles warmup on/off or changes the plan for a specific sender
func (s *Server) handleUpdateWarmup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	var req struct {
		Enabled bool   `json:"enabled"`
		Plan    string `json:"plan"` // "standard", "conservative", "aggressive"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	// Update DB fields
	sender.WarmupEnabled = req.Enabled
	if req.Enabled {
		sender.WarmupPlan = req.Plan
		// Safety: If enabling for the first time, start at Day 1
		if sender.WarmupDay == 0 {
			sender.WarmupDay = 1
			sender.WarmupLastUpdate = time.Now()
		}
	}

	if err := s.Store.UpdateSender(sender); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	// Apply config immediately to enforce new rate (don't wait for daily cron)
	go func() {
		snap, _ := core.LoadSnapshot(s.Store)
		core.ApplyKumoConfig(snap)
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
