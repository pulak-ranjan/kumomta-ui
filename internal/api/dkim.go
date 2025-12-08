package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// Request body for generating DKIM keys.
type dkimGenerateRequest struct {
	Domain    string `json:"domain"`               // required
	LocalPart string `json:"local_part,omitempty"` // if empty => all senders for this domain
}

// Response object for DKIM DNS records.
type dkimRecordDTO struct {
	Domain   string `json:"domain"`
	Selector string `json:"selector"`
	DNSName  string `json:"dns_name"`
	DNSValue string `json:"dns_value"`
}

// POST /api/dkim/generate
// Body:
//   { "domain": "example.com", "local_part": "editor" }
// or { "domain": "example.com" } to generate for all senders.
func (s *Server) handleGenerateDKIM(w http.ResponseWriter, r *http.Request) {
	var req dkimGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	req.Domain = strings.TrimSpace(req.Domain)
	req.LocalPart = strings.TrimSpace(req.LocalPart)

	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}

	d, err := s.Store.GetDomainByName(req.Domain)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load domain"})
		return
	}

	// If local_part specified -> generate for that sender only.
	if req.LocalPart != "" {
		found := false
		for _, snd := range d.Senders {
			if snd.LocalPart == req.LocalPart {
				found = true
				break
			}
		}
		if !found {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sender with that local_part not found for this domain"})
			return
		}

		if err := core.GenerateDKIMKey(d.Name, req.LocalPart); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate dkim key"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":      "ok",
			"domain":      d.Name,
			"local_part":  req.LocalPart,
			"message":     "dkim key generated",
		})
		return
	}

	// Otherwise, generate for all senders in this domain.
	if err := core.GenerateDKIMForDomainAllSenders(*d); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate dkim keys for all senders"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"domain":  d.Name,
		"message": "dkim keys generated for all senders",
	})
}

// GET /api/dkim/records
// Returns DNS-ready DKIM TXT records for all domains/senders that have pub keys.
func (s *Server) handleListDKIMRecords(w http.ResponseWriter, r *http.Request) {
	snap, err := core.LoadSnapshot(s.Store)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load snapshot"})
		return
	}

	recs, err := core.ListDKIMDNSRecords(snap)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list dkim records"})
		return
	}

	out := make([]dkimRecordDTO, 0, len(recs))
	for _, r := range recs {
		out = append(out, dkimRecordDTO{
			Domain:   r.Domain,
			Selector: r.Selector,
			DNSName:  r.DNSName,
			DNSValue: r.DNSValue,
		})
	}

	writeJSON(w, http.StatusOK, out)
}
