package api

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// DTOs
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
	HasDKIM        bool   `json:"has_dkim"`
	BounceUsername string `json:"bounce_username"`
}

func generateRandomPassword() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateRandomSuffix() string {
	b := make([]byte, 2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func checkDKIMExists(domain, selector string) bool {
	path := filepath.Join("/opt/kumomta/etc/dkim", domain, selector+".key")
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// ----------------------
// IMPORT HANDLER
// ----------------------

// POST /api/domains/import
// CSV: Domain, LocalPart, IP, Password, [BounceUser]
func (s *Server) handleImportSenders(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid csv format"})
		return
	}

	successCount := 0
	errorsLog := []string{}

	for i, row := range rows {
		if len(row) < 4 {
			errorsLog = append(errorsLog, fmt.Sprintf("Row %d: not enough columns", i+1))
			continue
		}

		rawDomain := strings.TrimSpace(row[0])
		rawLocal := strings.TrimSpace(row[1])
		rawIP := strings.TrimSpace(row[2])
		rawPass := strings.TrimSpace(row[3])
		rawBounceUser := ""
		if len(row) >= 5 {
			rawBounceUser = strings.TrimSpace(row[4])
		}

		if i == 0 && strings.ToLower(rawDomain) == "domain" {
			continue
		}

		if rawDomain == "" || rawLocal == "" || rawPass == "" {
			continue
		}

		// A. Domain
		d, err := s.Store.GetDomainByName(rawDomain)
		if err == store.ErrNotFound {
			newD := &models.Domain{
				Name:       rawDomain,
				MailHost:   "mail." + rawDomain,
				BounceHost: "bounce." + rawDomain,
			}
			if err := s.Store.CreateDomain(newD); err != nil {
				errorsLog = append(errorsLog, fmt.Sprintf("Row %d: failed to create domain %s", i+1, rawDomain))
				continue
			}
			d = newD
		} else if err != nil {
			continue
		}

		// B. IP
		if rawIP != "" {
			s.Store.CreateSystemIP(&models.SystemIP{Value: rawIP})
		}

		// C. Sender
		email := rawLocal + "@" + rawDomain
		sender := &models.Sender{
			DomainID:     d.ID,
			LocalPart:    rawLocal,
			Email:        email,
			IP:           rawIP,
			SMTPPassword: rawPass,
		}

		if err := s.Store.CreateSender(sender); err != nil {
			errorsLog = append(errorsLog, fmt.Sprintf("Row %d: failed to create sender %s", i+1, email))
			continue
		}

		// D. DKIM
		if err := core.GenerateDKIMKey(rawDomain, rawLocal); err != nil {
			fmt.Printf("Import Warning: DKIM failed for %s: %v\n", email, err)
		}

		// E. Bounce
		bounceUser := rawBounceUser
		if bounceUser == "" {
			// Generate safe unique name: b-{localpart}-{random}
			bounceUser = fmt.Sprintf("b-%s-%s", rawLocal, generateRandomSuffix())
		}

		bouncePass := generateRandomPassword()
		bounceHash, _ := bcrypt.GenerateFromPassword([]byte(bouncePass), bcrypt.DefaultCost)
		bounceAcc := &models.BounceAccount{
			Username:     bounceUser,
			PasswordHash: string(bounceHash),
			Domain:       rawDomain,
			Notes:        "Imported auto-create",
		}

		if err := s.Store.CreateBounceAccount(bounceAcc); err == nil {
			core.EnsureBounceAccount(*bounceAcc, bouncePass)
		} else {
			errorsLog = append(errorsLog, fmt.Sprintf("Row %d: Bounce user '%s' already exists", i+1, bounceUser))
		}

		successCount++
	}

	msg := fmt.Sprintf("Imported %d senders.", successCount)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"message": msg,
		"errors":  errorsLog,
	})
}

// ----------------------
// Existing Handlers
// ----------------------

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed"})
		return
	}
	bounces, _ := s.Store.ListBounceAccounts()
	bounceMap := make(map[string]string)
	for _, b := range bounces {
		// Key by notes is fuzzy but works for our auto-generated ones
		if strings.Contains(b.Notes, "Imported") || strings.Contains(b.Notes, "Auto-created") {
			bounceMap[b.Username] = b.Notes
		}
	}

	out := make([]domainDTO, 0, len(domains))
	for _, d := range domains {
		dDTO := domainToDTO(&d, false)
		dDTO.Senders = make([]senderDTO, 0, len(d.Senders))
		for _, sdr := range d.Senders {
			sDTO := senderToDTO(&sdr)
			sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
			// Check if we have a bounce user starting with b-{local}
			prefix := fmt.Sprintf("b-%s", sdr.LocalPart)
			// Try to find matching bounce in map by iterating. 
			// For perfect match logic, we need to store sender ID in bounce, but we don't.
			// So we search for prefix match in all known bounces.
			for bUser := range bounceMap {
				if strings.HasPrefix(bUser, prefix) {
					sDTO.BounceUsername = bUser
					break
				}
			}
			dDTO.Senders = append(dDTO.Senders, sDTO)
		}
		out = append(out, dDTO)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetDomain(w http.ResponseWriter, r *http.Request) {
	id, _ := parseUintParam(chi.URLParam(r, "domainID"))
	d, _ := s.Store.GetDomainByID(id)
	
	dDTO := domainToDTO(d, false)
	dDTO.Senders = make([]senderDTO, 0, len(d.Senders))
	bounces, _ := s.Store.ListBounceAccounts()
	
	for _, sdr := range d.Senders {
		sDTO := senderToDTO(&sdr)
		sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
		prefix := fmt.Sprintf("b-%s", sdr.LocalPart)
		for _, b := range bounces {
			if strings.HasPrefix(b.Username, prefix) {
				sDTO.BounceUsername = b.Username
				break
			}
		}
		dDTO.Senders = append(dDTO.Senders, sDTO)
	}
	writeJSON(w, http.StatusOK, dDTO)
}

func (s *Server) handleSaveDomain(w http.ResponseWriter, r *http.Request) {
	var dto domainDTO
	json.NewDecoder(r.Body).Decode(&dto)
	if dto.Name == "" { return }
	
	if dto.ID == 0 {
		d := &models.Domain{Name: dto.Name, MailHost: dto.MailHost, BounceHost: dto.BounceHost}
		s.Store.CreateDomain(d)
		writeJSON(w, http.StatusOK, domainToDTO(d, false))
	} else {
		d, _ := s.Store.GetDomainByID(dto.ID)
		d.Name = dto.Name
		d.MailHost = dto.MailHost
		d.BounceHost = dto.BounceHost
		s.Store.UpdateDomain(d)
		writeJSON(w, http.StatusOK, domainToDTO(d, false))
	}
}

func (s *Server) handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, _ := parseUintParam(chi.URLParam(r, "domainID"))
	s.Store.DeleteDomain(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListSenders(w http.ResponseWriter, r *http.Request) {
	domainID, _ := parseUintParam(chi.URLParam(r, "domainID"))
	d, _ := s.Store.GetDomainByID(domainID)
	senders, _ := s.Store.ListSendersByDomain(domainID)
	bounces, _ := s.Store.ListBounceAccounts()
	
	out := make([]senderDTO, 0)
	for _, sdr := range senders {
		sDTO := senderToDTO(&sdr)
		sDTO.HasDKIM = checkDKIMExists(d.Name, sdr.LocalPart)
		prefix := fmt.Sprintf("b-%s", sdr.LocalPart)
		for _, b := range bounces {
			if strings.HasPrefix(b.Username, prefix) {
				sDTO.BounceUsername = b.Username
				break
			}
		}
		out = append(out, sDTO)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSaveSender(w http.ResponseWriter, r *http.Request) {
	domainID, _ := parseUintParam(chi.URLParam(r, "domainID"))
	var dto senderDTO
	json.NewDecoder(r.Body).Decode(&dto)
	dto.DomainID = domainID
	
	if dto.ID == 0 {
		d, _ := s.Store.GetDomainByID(domainID)
		email := dto.Email
		if email == "" { email = dto.LocalPart + "@" + d.Name }
		
		sdr := &models.Sender{
			DomainID: domainID, LocalPart: dto.LocalPart, Email: email, 
			IP: dto.IP, SMTPPassword: dto.SMTPPassword,
		}
		s.Store.CreateSender(sdr)
		
		core.GenerateDKIMKey(d.Name, sdr.LocalPart)
		
		// Uniqueness fix with random suffix
		bounceUser := fmt.Sprintf("b-%s-%s", sdr.LocalPart, generateRandomSuffix())
		bouncePass := generateRandomPassword()
		hash, _ := bcrypt.GenerateFromPassword([]byte(bouncePass), bcrypt.DefaultCost)
		
		b := &models.BounceAccount{
			Username: bounceUser, PasswordHash: string(hash), Domain: d.Name,
			Notes: "Auto-created for " + email,
		}
		s.Store.CreateBounceAccount(b)
		core.EnsureBounceAccount(*b, bouncePass)
		
		resp := senderToDTO(sdr)
		resp.HasDKIM = true
		resp.BounceUsername = bounceUser
		writeJSON(w, http.StatusOK, resp)
	} else {
		// Update
		sdr, _ := s.Store.GetSenderByID(dto.ID)
		sdr.LocalPart = dto.LocalPart
		sdr.Email = dto.Email
		sdr.IP = dto.IP
		sdr.SMTPPassword = dto.SMTPPassword
		s.Store.UpdateSender(sdr)
		writeJSON(w, http.StatusOK, senderToDTO(sdr))
	}
}

func (s *Server) handleDeleteSenderByID(w http.ResponseWriter, r *http.Request) {
	id, _ := parseUintParam(chi.URLParam(r, "senderID"))
	s.Store.DeleteSender(id)
	writeJSON(w, http.StatusOK, map[string]string{"status":"deleted"})
}

func domainToDTO(d *models.Domain, includeSenders bool) domainDTO {
	dto := domainDTO{ID: d.ID, Name: d.Name, MailHost: d.MailHost, BounceHost: d.BounceHost}
	return dto
}
func senderToDTO(s *models.Sender) senderDTO {
	return senderDTO{ID: s.ID, DomainID: s.DomainID, LocalPart: s.LocalPart, Email: s.Email, IP: s.IP}
}
func parseUintParam(s string) (uint, error) {
	v, err := strconv.Atoi(s)
	return uint(v), err
}
