package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// GET /api/dmarc/{domainID}
func (s *Server) handleGetDMARC(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "domainID")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	record := core.GenerateDMARCRecord(domain)
	writeJSON(w, http.StatusOK, record)
}

// POST /api/dmarc/{domainID}
func (s *Server) handleSetDMARC(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "domainID")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	var req struct {
		Policy     string `json:"policy"`
		Rua        string `json:"rua"`
		Ruf        string `json:"ruf"`
		Percentage int    `json:"percentage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Validate policy
	if req.Policy != "" && !core.ValidateDMARCPolicy(req.Policy) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy (use: none, quarantine, reject)"})
		return
	}

	// Update domain
	if req.Policy != "" {
		domain.DMARCPolicy = req.Policy
	}
	domain.DMARCRua = req.Rua
	domain.DMARCRuf = req.Ruf

	if req.Percentage > 0 && req.Percentage <= 100 {
		domain.DMARCPercentage = req.Percentage
	} else if req.Percentage == 0 {
		domain.DMARCPercentage = 100 // Default
	}

	if err := s.Store.UpdateDomain(domain); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update domain"})
		return
	}

	// Return generated record
	record := core.GenerateDMARCRecord(domain)
	writeJSON(w, http.StatusOK, record)
}

// GET /api/dns/{domainID}
func (s *Server) handleGetAllDNS(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "domainID")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain id"})
		return
	}

	domain, err := s.Store.GetDomainByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	settings, _ := s.Store.GetSettings()
	mainIP := ""
	if settings != nil {
		mainIP = settings.MainServerIP
	}

	// 1. Expected Records
	snap, _ := core.LoadSnapshot(s.Store)
	generated := core.GenerateAllDNSRecords(domain, mainIP, snap)

	// 2. Live Records
	live, _ := core.LookupLiveDNS(domain)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"generated": generated,
		"live":      live,
	})
}
