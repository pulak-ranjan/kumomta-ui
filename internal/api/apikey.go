package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// GET /api/keys
// List all active API keys
func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	var keys []models.APIKey
	if err := s.Store.DB.Find(&keys).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// POST /api/keys
// Create a new API Key
func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"` // e.g. "relay,verify"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	// Generate a secure random key
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "crypto error"})
		return
	}
	keyStr := "kumo_" + hex.EncodeToString(bytes)

	apiKey := &models.APIKey{
		Name:      req.Name,
		Key:       keyStr,
		Scopes:    req.Scopes,
		CreatedAt: time.Now(),
	}

	if err := s.Store.DB.Create(apiKey).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save key"})
		return
	}

	writeJSON(w, http.StatusCreated, apiKey)
}

// DELETE /api/keys/{id}
// Revoke an API Key
func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	if err := s.Store.DB.Delete(&models.APIKey{}, id).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
