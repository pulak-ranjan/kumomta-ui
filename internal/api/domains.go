package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// DTOs for Domains and Senders as seen by the API/UI.

type domainDTO struct {
	ID         uint        `json:"id"`
	Name       string      `json:"name"`
	MailHost   string      `json:"mail_host"`
	BounceHost string      `json:"bounce_host"`
	Senders    []senderDTO `json:"senders"`
}

type senderDTO struct {
	ID           uint   `json:"id"`
	DomainID     uint   `json:"domain_id"`
	LocalPart    string `json:"local_part"`
	Email        string `json:"email"`
	IP           string `json:"ip"`
	SMTPPassword string `json:"smtp_password"`
}

// ----------------------
// Domain Handlers
// ----------------------

// GET /api/domains
func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list domains"})
		return
	}

	// Initialize as empty slice, not nil - this ensures JSON returns [] not null
	out := make([]domainDTO, 0, len(domains))
	for _, d := range domains {
		out = append(out, domainToDTO(&d, true))
	}

	writeJSON(w, http.StatusOK, out)
}

// GET /api/domains/{domainID}
func (s *Server) handleGetDomain(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(chi.URLParam(r, "domainID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	d, err := s.Store.GetDomainByID(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load domain"})
		return
	}

	writeJSON(w, http.StatusOK, domainToDTO(d, true))
}

// POST /api/domains
// If dto.id == 0 -> create, else update.
func (s *Server) handleSaveDomain(w http.ResponseWriter, r *http.Request) {
	var dto domainDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// VALIDATION FIX: Ensure name is not empty
	if dto.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain name is required"})
		return
	}

	// Create
	if dto.ID == 0 {
		d := &models.Domain{
			Name:       dto.Name,
			MailHost:   dto.MailHost,
			BounceHost: dto.BounceHost,
		}
		if err := s.Store.CreateDomain(d); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create domain"})
			return
		}
		writeJSON(w, http.StatusOK, domainToDTO(d, false))
		return
	}

	// Update
	d, err := s.Store.GetDomainByID(dto.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load domain for update"})
		return
	}

	d.Name = dto.Name
	d.MailHost = dto.MailHost
	d.BounceHost = dto.BounceHost

	if err := s.Store.UpdateDomain(d); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update domain"})
		return
	}

	writeJSON(w, http.StatusOK, domainToDTO(d, false))
}

// DELETE /api/domains/{domainID}
func (s *Server) handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(chi.URLParam(r, "domainID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	if err := s.Store.DeleteDomain(id); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete domain"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ----------------------
// Sender Handlers
// ----------------------

// GET /api/domains/{domainID}/senders
func (s *Server) handleListSenders(w http.ResponseWriter, r *http.Request) {
	domainID, err := parseUintParam(chi.URLParam(r, "domainID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	senders, err := s.Store.ListSendersByDomain(domainID)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list senders"})
		return
	}

	// Initialize as empty slice, not nil
	out := make([]senderDTO, 0, len(senders))
	for _, sdr := range senders {
		out = append(out, senderToDTO(&sdr))
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/domains/{domainID}/senders
// If dto.id == 0 -> create; else update.
func (s *Server) handleSaveSender(w http.ResponseWriter, r *http.Request) {
	domainID, err := parseUintParam(chi.URLParam(r, "domainID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	var dto senderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Force domainID from URL
	dto.DomainID = domainID

	// Create
	if dto.ID == 0 {
		email := dto.Email
		if email == "" && dto.LocalPart != "" {
			// we can derive email if we know domain
			if d, err := s.Store.GetDomainByID(domainID); err == nil {
				email = dto.LocalPart + "@" + d.Name
			}
		}

		sdr := &models.Sender{
			DomainID:     dto.DomainID,
			LocalPart:    dto.LocalPart,
			Email:        email,
			IP:           dto.IP,
			SMTPPassword: dto.SMTPPassword,
		}

		if err := s.Store.CreateSender(sdr); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create sender"})
			return
		}
		writeJSON(w, http.StatusOK, senderToDTO(sdr))
		return
	}

	// Update
	sdr, err := s.Store.GetSenderByID(dto.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load sender for update"})
		return
	}

	sdr.DomainID = dto.DomainID
	sdr.LocalPart = dto.LocalPart
	sdr.Email = dto.Email
	sdr.IP = dto.IP
	sdr.SMTPPassword = dto.SMTPPassword

	if err := s.Store.UpdateSender(sdr); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update sender"})
		return
	}

	writeJSON(w, http.StatusOK, senderToDTO(sdr))
}

// DELETE /api/senders/{senderID}
func (s *Server) handleDeleteSenderByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(chi.URLParam(r, "senderID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sender id"})
		return
	}

	if err := s.Store.DeleteSender(id); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete sender"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ----------------------
// Helpers
// ----------------------

func domainToDTO(d *models.Domain, includeSenders bool) domainDTO {
	dto := domainDTO{
		ID:         d.ID,
		Name:       d.Name,
		MailHost:   d.MailHost,
		BounceHost: d.BounceHost,
		Senders:    make([]senderDTO, 0), // Initialize as empty slice, not nil
	}

	if includeSenders && len(d.Senders) > 0 {
		for _, sdr := range d.Senders {
			dto.Senders = append(dto.Senders, senderToDTO(&sdr))
		}
	}

	return dto
}

// SECURITY FIX: Never return the password in the API response.
func senderToDTO(sdr *models.Sender) senderDTO {
	return senderDTO{
		ID:        sdr.ID,
		DomainID:  sdr.DomainID,
		LocalPart: sdr.LocalPart,
		Email:     sdr.Email,
		IP:        sdr.IP,
		// SMTPPassword: sdr.SMTPPassword, <--- REMOVED FOR SECURITY
	}
}

func parseUintParam(raw string) (uint, error) {
	i, err := strconv.Atoi(raw)
	if err != nil || i < 0 {
		return 0, err
	}
	return uint(i), nil
}
