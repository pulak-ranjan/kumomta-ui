package api

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// Transparent 1x1 GIF
var pixelGIF, _ = base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")

type TrackingHandler struct {
	Store *store.Store
}

func NewTrackingHandler(st *store.Store) *TrackingHandler {
	return &TrackingHandler{Store: st}
}

// GET /api/track/open/{recipient_id}
func (h *TrackingHandler) HandleTrackOpen(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	if id > 0 {
		go h.recordOpen(uint(id), r)
	}

	// Return transparent pixel
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(pixelGIF)
}

// GET /api/track/click/{recipient_id}?url=...
func (h *TrackingHandler) HandleTrackClick(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)
	targetURL := r.URL.Query().Get("url")

	// Security: Prevent Open Redirect to non-HTTP protocols (e.g. javascript:)
	// TODO: Ideally verify domain against allowlist or sign the URL.
	if targetURL == "" || (!strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://")) {
		targetURL = "/" // Fallback
	}

	if id > 0 {
		go h.recordClick(uint(id))
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (h *TrackingHandler) recordOpen(id uint, r *http.Request) {
	var recip models.CampaignRecipient
	if err := h.Store.DB.First(&recip, id).Error; err != nil {
		return
	}

	// Update Recipient
	now := time.Now()
	if recip.OpenedAt == nil {
		recip.OpenedAt = &now
		h.Store.DB.Save(&recip)

		// Increment Campaign Stats
		var camp models.Campaign
		if err := h.Store.DB.First(&camp, recip.CampaignID).Error; err == nil {
			camp.TotalOpens++
			h.Store.DB.Save(&camp)
		}

		// Update Contact Score (AI Superlead)
		if recip.ContactID > 0 {
			var contact models.Contact
			if err := h.Store.DB.First(&contact, recip.ContactID).Error; err == nil {
				contact.TotalOpens++
				contact.Score += 1 // +1 for open
				h.Store.DB.Save(&contact)
			}
		}
	}
}

func (h *TrackingHandler) recordClick(id uint) {
	var recip models.CampaignRecipient
	if err := h.Store.DB.First(&recip, id).Error; err != nil {
		return
	}

	now := time.Now()
	if recip.ClickedAt == nil {
		recip.ClickedAt = &now
		h.Store.DB.Save(&recip)

		// Increment Campaign Stats
		var camp models.Campaign
		if err := h.Store.DB.First(&camp, recip.CampaignID).Error; err == nil {
			camp.TotalClicks++
			h.Store.DB.Save(&camp)
		}

		// Update Contact Score
		if recip.ContactID > 0 {
			var contact models.Contact
			if err := h.Store.DB.First(&contact, recip.ContactID).Error; err == nil {
				contact.TotalClicks++
				contact.Score += 5 // +5 for click (higher intent)
				h.Store.DB.Save(&contact)
			}
		}
	}
}
