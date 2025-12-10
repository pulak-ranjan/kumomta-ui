package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

type testEmailRequest struct {
	SenderEmail string `json:"sender"`    // e.g. editor@domain.com
	Recipient   string `json:"recipient"` // e.g. mailtester@...
	Subject     string `json:"subject"`
	Body        string `json:"body"`
}

// POST /api/tools/send-test
func (s *Server) handleSendTestEmail(w http.ResponseWriter, r *http.Request) {
	var req testEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.SenderEmail == "" || req.Recipient == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sender and recipient required"})
		return
	}

	// 1. DYNAMIC LOOKUP: Find the sender in the database to get real config
	var sender models.Sender
	if err := s.Store.DB.Preload("Domain").Where("email = ?", req.SenderEmail).First(&sender).Error; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Sender '%s' not found in KumoMTA UI. Please add it to Domains first.", req.SenderEmail),
		})
		return
	}

	// 2. CONSTRUCT HELO: Use the configured naming convention
	helo := fmt.Sprintf("mail.%s", sender.Domain.Name) 
	if sender.LocalPart != "" {
		helo = fmt.Sprintf("%s.%s", sender.LocalPart, sender.Domain.Name)
	}

	// 3. EXECUTE SWAKS
	// We connect to localhost:25. KumoMTA's init.lua uses the 'MAIL FROM' to map to the correct source IP.
	args := []string{
		"--to", req.Recipient,
		"--from", sender.Email,
		"--server", "127.0.0.1",
		"--port", "25",
		"--helo", helo,
		"--header", "Subject: " + req.Subject,
		"--header", "X-Kumo-Test: True", // Header to identify test traffic
		"--body", req.Body,
		"--hide-all",
	}

	cmdStr := fmt.Sprintf("swaks %s", strings.Join(args, " "))
	
	cmd := exec.Command("swaks", args...)
	output, err := cmd.CombinedOutput()

	response := map[string]string{
		"status":      "sent",
		"sender_ip":   sender.IP, // The IP KumoMTA *should* use (for verification)
		"used_helo":   helo,
		"smtp_output": string(output),
		"command":     cmdStr,
	}

	if err != nil {
		response["status"] = "failed"
		response["error"] = err.Error()
		writeJSON(w, http.StatusInternalServerError, response)
		return
	}

	writeJSON(w, http.StatusOK, response)
}
