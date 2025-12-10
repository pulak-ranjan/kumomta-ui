package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// GET /api/stats/domains
// Returns aggregated stats for all domains over the requested period (default 7 days).
func (s *Server) handleGetDomainStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	// This calls the optimized, parallelized core function
	stats, err := core.GetAllDomainsStats(days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GET /api/stats/domains/{domain}
// Returns detailed daily stats for a specific domain.
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
// Returns a high-level summary (Sent/Bounced/Delivered) for the Dashboard.
func (s *Server) handleGetStatsSummary(w http.ResponseWriter, r *http.Request) {
	// Get today's stats (1 day) for the quick view
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
// Triggers a manual parsing of logs to update the database cache.
func (s *Server) handleRefreshStats(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
		hours = h
	}

	// This now uses the Zstd-capable, parallel parser
	if err := core.ParseKumoLogs(s.Store, hours); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to parse logs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}
