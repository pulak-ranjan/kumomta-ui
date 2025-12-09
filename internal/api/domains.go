package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// ----------------------
// Domains
// ----------------------

// GET /api/domains
func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list domains"})
		return
	}
	if domains == nil {
		domains = []models.Domain{}
	}
	writeJSON(w, http.StatusOK, domains)
}

// POST /api/domains
func (s *Server) handleCreateDomain(w http.ResponseWriter, r *http.Request) {
	var d models.Domain
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if d.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	// Set defaults
	if d.MailHost == "" {
		d.MailHost = "mail." + d.Name
	}
	if d.BounceHost == "" {
		d.BounceHost = "bounce." + d.Name
	}
	if d.DMARCPolicy == "" {
		d.DMARCPolicy = "none"
	}
	if d.DMARCPercentage == 0 {
		d.DMARCPercentage = 100
	}

	if err := s.Store.CreateDomain(&d); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create domain"})
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

// GET /api/domains/{id}
func (s *Server) handleGetDomain(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(id))
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get domain"})
		return
	}

	writeJSON(w, http.StatusOK, domain)
}

// PUT /api/domains/{id}
func (s *Server) handleUpdateDomain(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	var update models.Domain
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if update.Name != "" {
		domain.Name = update.Name
	}
	if update.MailHost != "" {
		domain.MailHost = update.MailHost
	}
	if update.BounceHost != "" {
		domain.BounceHost = update.BounceHost
	}
	if update.DMARCPolicy != "" {
		domain.DMARCPolicy = update.DMARCPolicy
	}
	if update.DMARCRua != "" {
		domain.DMARCRua = update.DMARCRua
	}
	if update.DMARCRuf != "" {
		domain.DMARCRuf = update.DMARCRuf
	}
	if update.DMARCPercentage > 0 {
		domain.DMARCPercentage = update.DMARCPercentage
	}

	if err := s.Store.UpdateDomain(domain); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update domain"})
		return
	}

	writeJSON(w, http.StatusOK, domain)
}

// DELETE /api/domains/{id}
func (s *Server) handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteDomain(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete domain"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ----------------------
// Senders
// ----------------------

// GET /api/domains/{domainID}/senders
func (s *Server) handleListSenders(w http.ResponseWriter, r *http.Request) {
	domainIDStr := chi.URLParam(r, "domainID")
	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	senders, err := s.Store.ListSendersByDomain(uint(domainID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list senders"})
		return
	}
	if senders == nil {
		senders = []models.Sender{}
	}

	writeJSON(w, http.StatusOK, senders)
}

// POST /api/domains/{domainID}/senders
func (s *Server) handleCreateSender(w http.ResponseWriter, r *http.Request) {
	domainIDStr := chi.URLParam(r, "domainID")
	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	// Verify domain exists
	domain, err := s.Store.GetDomainByID(uint(domainID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	var snd models.Sender
	if err := json.NewDecoder(r.Body).Decode(&snd); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	snd.DomainID = uint(domainID)
	if snd.LocalPart != "" && snd.Email == "" {
		snd.Email = snd.LocalPart + "@" + domain.Name
	}

	if err := s.Store.CreateSender(&snd); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create sender"})
		return
	}

	writeJSON(w, http.StatusCreated, snd)
}

// GET /api/senders/{id}
func (s *Server) handleGetSender(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	writeJSON(w, http.StatusOK, sender)
}

// PUT /api/senders/{id}
func (s *Server) handleUpdateSender(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	var update models.Sender
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if update.LocalPart != "" {
		sender.LocalPart = update.LocalPart
	}
	if update.Email != "" {
		sender.Email = update.Email
	}
	if update.IP != "" {
		sender.IP = update.IP
	}
	if update.SMTPPassword != "" {
		sender.SMTPPassword = update.SMTPPassword
	}

	if err := s.Store.UpdateSender(sender); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update sender"})
		return
	}

	writeJSON(w, http.StatusOK, sender)
}

// DELETE /api/senders/{id}
func (s *Server) handleDeleteSender(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteSender(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete sender"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/domains/{domainID}/senders/{id}/setup
func (s *Server) handleSetupSender(w http.ResponseWriter, r *http.Request) {
	domainIDStr := chi.URLParam(r, "domainID")
	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	senderIDStr := chi.URLParam(r, "id")
	senderID, err := strconv.ParseUint(senderIDStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sender id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(domainID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	sender, err := s.Store.GetSenderByID(uint(senderID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	// 1. Generate DKIM key
	dkimErr := core.GenerateDKIM(domain.Name, sender.LocalPart)

	// 2. Create bounce account
	bounceUser := "b-" + sender.LocalPart
	bounceErr := core.CreateBounceAccount(bounceUser, domain.Name, s.Store)

	result := map[string]interface{}{
		"dkim_generated":   dkimErr == nil,
		"bounce_created":   bounceErr == nil,
		"bounce_user":      bounceUser + "@" + domain.Name,
		"selector":         sender.LocalPart,
	}

	if dkimErr != nil {
		result["dkim_error"] = dkimErr.Error()
	}
	if bounceErr != nil {
		result["bounce_error"] = bounceErr.Error()
	}

	writeJSON(w, http.StatusOK, result)
}
