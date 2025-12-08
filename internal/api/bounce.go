package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type bounceDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	Notes    string `json:"notes"`
}

// GET /api/bounces
func (s *Server) handleListBounceAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListBounceAccounts()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list bounce accounts"})
		return
	}

	out := make([]bounceDTO, 0, len(list))
	for _, b := range list {
		out = append(out, bounceDTO{
			ID:       b.ID,
			Username: b.Username,
			Password: b.Password,
			Domain:   b.Domain,
			Notes:    b.Notes,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/bounces
// If dto.id == 0 => create; else update.
func (s *Server) handleSaveBounceAccount(w http.ResponseWriter, r *http.Request) {
	var dto bounceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	dto.Username = strings.TrimSpace(dto.Username)
	dto.Password = strings.TrimSpace(dto.Password)
	dto.Domain = strings.TrimSpace(dto.Domain)

	if dto.Username == "" || dto.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}

	if dto.ID == 0 {
		// Create
		b := &models.BounceAccount{
			Username: dto.Username,
			Password: dto.Password,
			Domain:   dto.Domain,
			Notes:    dto.Notes,
		}
		if err := s.Store.CreateBounceAccount(b); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create bounce account"})
			return
		}
		dto.ID = b.ID
		writeJSON(w, http.StatusOK, dto)
		return
	}

	// Update
	existing, err := s.Store.GetBounceAccountByID(dto.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "bounce account not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load bounce account"})
		return
	}

	existing.Username = dto.Username
	existing.Password = dto.Password
	existing.Domain = dto.Domain
	existing.Notes = dto.Notes

	if err := s.Store.UpdateBounceAccount(existing); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update bounce account"})
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

// DELETE /api/bounces/{bounceID}
func (s *Server) handleDeleteBounceAccount(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "bounceID")
	id, err := strconv.Atoi(raw)
	if err != nil || id < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bounce id"})
		return
	}

	if err := s.Store.DeleteBounceAccount(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete bounce account"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/bounces/apply
// Ensures all bounce accounts exist at OS level with Maildir.
func (s *Server) handleApplyBounceAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListBounceAccounts()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list bounce accounts"})
		return
	}

	if err := core.ApplyAllBounceAccounts(list); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
