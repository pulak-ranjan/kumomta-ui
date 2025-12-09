package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// GET /api/stats/domains
func (s *Server) handleGetDomainStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	stats, err := core.GetAllDomainsStats(days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GET /api/stats/domains/{domain}
func (s *Server) handleGetSingleDomainStats(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain required"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	stats, err := core.GetDomainStatsFromLogs(domain, days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GET /api/stats/summary
func (s *Server) handleGetStatsSummary(w http.ResponseWriter, r *http.Request) {
	// Get today's stats
	stats, err := core.GetAllDomainsStats(1)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	summary := struct {
		TotalSent      int64   `json:"total_sent"`
		TotalDelivered int64   `json:"total_delivered"`
		TotalBounced   int64   `json:"total_bounced"`
		TotalDeferred  int64   `json:"total_deferred"`
		DeliveryRate   float64 `json:"delivery_rate"`
		BounceRate     float64 `json:"bounce_rate"`
		DomainsActive  int     `json:"domains_active"`
	}{}

	for _, domainStats := range stats {
		for _, day := range domainStats {
			summary.TotalSent += day.Sent
			summary.TotalDelivered += day.Delivered
			summary.TotalBounced += day.Bounced
			summary.TotalDeferred += day.Deferred
		}
	}

	summary.DomainsActive = len(stats)

	if summary.TotalSent > 0 {
		summary.DeliveryRate = float64(summary.TotalDelivered) / float64(summary.TotalSent) * 100
		summary.BounceRate = float64(summary.TotalBounced) / float64(summary.TotalSent) * 100
	}

	writeJSON(w, http.StatusOK, summary)
}

// POST /api/stats/refresh
func (s *Server) handleRefreshStats(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
		hours = h
	}

	if err := core.ParseKumoLogs(s.Store, hours); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to parse logs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}
