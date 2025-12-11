package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type bounceDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // plain, only from client to set/update; never returned
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
			Password: "", // never return password
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

	if dto.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	// CREATE
	if dto.ID == 0 {
		if dto.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password is required for new bounce account"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}

		b := &models.BounceAccount{
			Username:     dto.Username,
			PasswordHash: string(hash),
			Domain:       dto.Domain,
			Notes:        dto.Notes,
		}

		if err := s.Store.CreateBounceAccount(b); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create bounce account"})
			return
		}

		// Apply to system immediately (create user + Maildir + set password)
		if err := core.EnsureBounceAccount(*b, dto.Password); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "created in DB but failed to create system user: " + err.Error()})
			return
		}

		dto.ID = b.ID
		dto.Password = "" // do not echo password back
		writeJSON(w, http.StatusOK, dto)
		return
	}

	// UPDATE
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
	existing.Domain = dto.Domain
	existing.Notes = dto.Notes

	// If a new password is provided, re-hash and update system password.
	var plainToApply string
	if dto.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash new password"})
			return
		}
		existing.PasswordHash = string(hash)
		plainToApply = dto.Password
	}

	if err := s.Store.UpdateBounceAccount(existing); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update bounce account"})
		return
	}

	// Ensure system user + Maildir; update password only if provided
	if err := core.EnsureBounceAccount(*existing, plainToApply); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "updated in DB but failed to apply to system: " + err.Error()})
		return
	}

	dto.Password = "" // never return password
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

	// 1. Get the account first so we know the username
	account, err := s.Store.GetBounceAccountByID(uint(id))
	if err != nil {
		// If not found in DB, just return success or not found
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	// 2. Delete from System (OS)
	if err := core.RemoveSystemUser(account.Username); err != nil {
		// Log error but continue to delete from DB so they aren't stuck
		s.Store.LogError(err)
	}

	// 3. Delete from Database
	if err := s.Store.DeleteBounceAccount(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete bounce account"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/bounces/apply
// Ensures all bounce accounts exist at OS level with Maildir, without changing passwords.
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
