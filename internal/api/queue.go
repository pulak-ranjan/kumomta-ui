package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// GET /api/queue
func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}

	messages, err := core.GetQueueMessages(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read queue"})
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// GET /api/queue/stats
func (s *Server) handleGetQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := core.GetQueueStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get queue stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// DELETE /api/queue/{id}
func (s *Server) handleDeleteQueueMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if err := core.DeleteQueueMessage(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete message"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/queue/flush
func (s *Server) handleFlushQueue(w http.ResponseWriter, r *http.Request) {
	if err := core.FlushQueue(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to flush queue"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "flushed"})
}
