package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
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
	ID             uint   `json:"id"`
	DomainID       uint   `json:"domain_id"`
	LocalPart      string `json:"local_part"`
	Email          string `json:"email"`
	IP             string `json:"ip"`
	SMTPPassword   string `json:"smtp_password"`
	HasDKIM        bool   `json:"has_dkim"`        // <--- NEW
	BounceUsername string `json:"bounce_username"` // <--- NEW
}

// Helper to generate a random password for bounce accounts
func generateRandomPassword() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Helper to check if DKIM key exists on disk
func checkDKIMExists(domain, selector string) bool {
	// Path: /opt/kumomta/etc/dkim/<domain>/<selector>.key
	path := filepath.Join("/opt/kumomta/etc/dkim", domain, selector+".key")
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
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

	// Pre-fetch all bounce accounts to map for O(1) lookup
	// Map: "b-localpart" -> true (or strict matching logic)
	// Actually, we need to find which bounce account matches the sender.
	// Our convention is b-localpart. Let's build a map of usernames.
	bounces, _ := s.Store.ListBounceAccounts()
	bounceMap := make(map[string]bool)
	for _, b := range bounces {
		bounceMap[b.Username] = true
	}

	out := make([]domainDTO, 0, len(domains))
	for _, d := range domains {
		dDTO := domainToDTO(&d, false) // get base without senders first
		dDTO.Senders = make([]senderDTO, 0, len(d.Senders))

		for _, sdr := range d.Senders {
			sDTO := senderToDTO(&sdr)

			// 1. Check DKIM
			if sdr.LocalPart != "" {
				sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
			}

			// 2. Check Bounce (Convention: b-localpart)
			expectedBounce := fmt.Sprintf("b-%s", sdr.LocalPart)
			if bounceMap[expectedBounce] {
				sDTO.BounceUsername = expectedBounce
			}

			dDTO.Senders = append(dDTO.Senders, sDTO)
		}
		out = append(out, dDTO)
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

	// Prepare single DTO with status checks
	dDTO := domainToDTO(d, false)
	dDTO.Senders = make([]senderDTO, 0, len(d.Senders))
	
	// Bounce map
	bounces, _ := s.Store.ListBounceAccounts()
	bounceMap := make(map[string]bool)
	for _, b := range bounces {
		bounceMap[b.Username] = true
	}

	for _, sdr := range d.Senders {
		sDTO := senderToDTO(&sdr)
		if sdr.LocalPart != "" {
			sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
		}
		expectedBounce := fmt.Sprintf("b-%s", sdr.LocalPart)
		if bounceMap[expectedBounce] {
			sDTO.BounceUsername = expectedBounce
		}
		dDTO.Senders = append(dDTO.Senders, sDTO)
	}

	writeJSON(w, http.StatusOK, dDTO)
}

// POST /api/domains
func (s *Server) handleSaveDomain(w http.ResponseWriter, r *http.Request) {
	var dto domainDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

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

	// Need domain name for DKIM check
	d, err := s.Store.GetDomainByID(domainID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load domain info"})
		return
	}

	senders, err := s.Store.ListSendersByDomain(domainID)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list senders"})
		return
	}

	// Bounce map
	bounces, _ := s.Store.ListBounceAccounts()
	bounceMap := make(map[string]bool)
	for _, b := range bounces {
		bounceMap[b.Username] = true
	}

	out := make([]senderDTO, 0, len(senders))
	for _, sdr := range senders {
		sDTO := senderToDTO(&sdr)
		if sdr.LocalPart != "" {
			sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
		}
		expectedBounce := fmt.Sprintf("b-%s", sdr.LocalPart)
		if bounceMap[expectedBounce] {
			sDTO.BounceUsername = expectedBounce
		}
		out = append(out, sDTO)
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/domains/{domainID}/senders
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

	dto.DomainID = domainID

	// Create
	if dto.ID == 0 {
		d, err := s.Store.GetDomainByID(domainID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}

		email := dto.Email
		if email == "" && dto.LocalPart != "" {
			email = dto.LocalPart + "@" + d.Name
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

		// --- AUTOMATION START ---
		
		// 1. Auto-Generate DKIM Key
		if sdr.LocalPart != "" {
			if err := core.GenerateDKIMKey(d.Name, sdr.LocalPart); err != nil {
				s.Store.LogError(err)
			}
		}

		// 2. Auto-Create Bounce Account (System User)
		bounceUser := fmt.Sprintf("b-%s", sdr.LocalPart)
		bouncePass := generateRandomPassword()
		bounceHash, _ := bcrypt.GenerateFromPassword([]byte(bouncePass), bcrypt.DefaultCost)

		bounceAcc := &models.BounceAccount{
			Username:     bounceUser,
			PasswordHash: string(bounceHash),
			Domain:       d.Name,
			Notes:        fmt.Sprintf("Auto-created for sender %s", sdr.Email),
		}

		// Save bounce to DB
		if err := s.Store.CreateBounceAccount(bounceAcc); err == nil {
			// Apply to system (useradd)
			core.EnsureBounceAccount(*bounceAcc, bouncePass)
		} else {
			// Duplicate? Ignore logic for now, just log
			s.Store.LogError(err)
		}
		// --- AUTOMATION END ---

		// Return DTO with checks
		respDTO := senderToDTO(sdr)
		respDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
		// Check bounce
		if _, err := s.Store.GetBounceAccountByID(bounceAcc.ID); err == nil {
			respDTO.BounceUsername = bounceUser
		}
		
		writeJSON(w, http.StatusOK, respDTO)
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

	// We don't auto-regen DKIM on update to allow manual management
	// But we do return the status
	d, _ := s.Store.GetDomainByID(sdr.DomainID)
	
	respDTO := senderToDTO(sdr)
	if d != nil {
		respDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
	}
	// Check bounce simply by convention for update
	bName := fmt.Sprintf("b-%s", sdr.LocalPart)
	// Just return the convention if we assume it exists, but proper check is better
	// For speed, let's just return the DTO. The list view will have the full map check.
	
	writeJSON(w, http.StatusOK, respDTO)
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
		Senders:    make([]senderDTO, 0),
	}

	if includeSenders && len(d.Senders) > 0 {
		for _, sdr := range d.Senders {
			dto.Senders = append(dto.Senders, senderToDTO(&sdr))
		}
	}

	return dto
}

func senderToDTO(sdr *models.Sender) senderDTO {
	return senderDTO{
		ID:        sdr.ID,
		DomainID:  sdr.DomainID,
		LocalPart: sdr.LocalPart,
		Email:     sdr.Email,
		IP:        sdr.IP,
		// SMTPPassword omitted
	}
}

func parseUintParam(raw string) (uint, error) {
	i, err := strconv.Atoi(raw)
	if err != nil || i < 0 {
		return 0, err
	}
	return uint(i), nil
}
