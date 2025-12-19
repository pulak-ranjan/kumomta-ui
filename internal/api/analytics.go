package api

import (
	"net/http"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type AnalyticsHandler struct {
	Store *store.Store
}

func NewAnalyticsHandler(st *store.Store) *AnalyticsHandler {
	return &AnalyticsHandler{Store: st}
}

// GET /api/analytics/top-leads
func (h *AnalyticsHandler) GetTopLeads(w http.ResponseWriter, r *http.Request) {
	var contacts []models.Contact

	// Top 20 contacts by lead score
	if err := h.Store.DB.Order("score desc").Limit(20).Find(&contacts).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	writeJSON(w, http.StatusOK, contacts)
}

// GET /api/analytics/campaign-summary
func (h *AnalyticsHandler) GetCampaignSummary(w http.ResponseWriter, r *http.Request) {
	// Aggregate stats across all campaigns
	var stats struct {
		TotalSent   int
		TotalOpens  int
		TotalClicks int
	}

	h.Store.DB.Model(&models.Campaign{}).Select("sum(total_sent) as total_sent, sum(total_opens) as total_opens, sum(total_clicks) as total_clicks").Scan(&stats)

	writeJSON(w, http.StatusOK, stats)
}
