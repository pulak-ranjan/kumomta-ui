package api

import (
	"encoding/csv"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// POST /api/import/csv
func (s *Server) handleCSVImport(w http.ResponseWriter, r *http.Request) {
	// Max 10MB
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Variable fields

	// Read header
	header, err := reader.Read()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid CSV"})
		return
	}

	// Map column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Required: domain
	domainIdx, hasDomain := colMap["domain"]
	if !hasDomain {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing 'domain' column"})
		return
	}

	// Optional columns
	senderIdx := colMap["sender"]
	localPartIdx := colMap["localpart"]
	ipIdx := colMap["ip"]

	var stats struct {
		DomainsCreated  int      `json:"domains_created"`
		SendersCreated  int      `json:"senders_created"`
		IPsAdded        int      `json:"ips_added"`
		BouncesCreated  int      `json:"bounces_created"`
		DKIMGenerated   int      `json:"dkim_generated"`
		Errors          []string `json:"errors"`
	}

	// Track created domains and senders
	domainCache := make(map[string]*models.Domain)
	processedSenders := make(map[string]bool)

	lineNum := 1
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			stats.Errors = append(stats.Errors, "line "+string(rune(lineNum))+": invalid format")
			continue
		}

		// Get domain
		if domainIdx >= len(record) {
			continue
		}
		domainName := strings.TrimSpace(record[domainIdx])
		if domainName == "" {
			continue
		}

		// Get or create domain
		domain, exists := domainCache[domainName]
		if !exists {
			existing, err := s.Store.GetDomainByName(domainName)
			if err != nil {
				// Create new domain
				domain = &models.Domain{
					Name:            domainName,
					MailHost:        "mail." + domainName,
					BounceHost:      "bounce." + domainName,
					DMARCPolicy:     "none",
					DMARCPercentage: 100,
				}
				if err := s.Store.CreateDomain(domain); err != nil {
					stats.Errors = append(stats.Errors, "failed to create domain: "+domainName)
					continue
				}
				stats.DomainsCreated++
			} else {
				domain = existing
			}
			domainCache[domainName] = domain
		}

		// Get sender/localpart
		localPart := ""
		if localPartIdx > 0 && localPartIdx < len(record) {
			localPart = strings.TrimSpace(record[localPartIdx])
		}
		if localPart == "" && senderIdx > 0 && senderIdx < len(record) {
			senderEmail := strings.TrimSpace(record[senderIdx])
			if parts := strings.Split(senderEmail, "@"); len(parts) == 2 {
				localPart = parts[0]
			} else {
				localPart = senderEmail
			}
		}

		if localPart == "" {
			continue // No sender to process
		}

		// Get IP
		ip := ""
		if ipIdx > 0 && ipIdx < len(record) {
			ip = strings.TrimSpace(record[ipIdx])
		}

		// Check if already processed
		senderKey := domainName + ":" + localPart
		if processedSenders[senderKey] {
			continue
		}
		processedSenders[senderKey] = true

		// Create sender
		sender := &models.Sender{
			DomainID:  domain.ID,
			LocalPart: localPart,
			Email:     localPart + "@" + domainName,
			IP:        ip,
		}
		if err := s.Store.CreateSender(sender); err != nil {
			stats.Errors = append(stats.Errors, "failed to create sender: "+sender.Email)
			continue
		}
		stats.SendersCreated++

		// Add IP to inventory if provided
		if ip != "" {
			ipModel := &models.SystemIP{
				Value:     ip,
				CreatedAt: time.Now(),
			}
			if err := s.Store.CreateSystemIP(ipModel); err == nil {
				stats.IPsAdded++
			}
		}

		// Generate DKIM - FIX: use GenerateDKIMKey
		if err := core.GenerateDKIMKey(domainName, localPart); err == nil {
			stats.DKIMGenerated++
		}

		// Create unique bounce account
		bounceUser := "b-" + localPart
		if err := core.CreateBounceAccount(bounceUser, domainName, s.Store); err == nil {
			stats.BouncesCreated++
		}
	}

	writeJSON(w, http.StatusOK, stats)
}
