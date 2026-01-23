package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time" // <--- 1. ADD THIS IMPORT

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

type testEmailRequest struct {
	SenderEmail string `json:"sender"`
	Recipient   string `json:"recipient"`
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

	// 1. DYNAMIC LOOKUP
	var sender models.Sender
	if err := s.Store.DB.Preload("Domain").Where("email = ?", req.SenderEmail).First(&sender).Error; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Sender '%s' not found. Please add to Domains first.", req.SenderEmail),
		})
		return
	}

	// 2. CONSTRUCT HELO
	helo := fmt.Sprintf("mail.%s", sender.Domain.Name)
	if sender.LocalPart != "" {
		helo = fmt.Sprintf("%s.%s", sender.LocalPart, sender.Domain.Name)
	}

	// 3. GENERATE VALID MESSAGE-ID
	// Format: <timestamp.nanoseconds@domain.com>
	msgID := fmt.Sprintf("<%d.%s@%s>", time.Now().Unix(), "test", sender.Domain.Name)

	// 4. EXECUTE SWAKS
	args := []string{
		"--to", req.Recipient,
		"--from", sender.Email,
		"--server", "127.0.0.1",
		"--port", "25",
		"--helo", helo,
		"--header", "Subject: " + req.Subject,
		"--header", "Message-Id: " + msgID, // <--- 2. PASS EXPLICIT ID HERE
		"--header", "X-Kumo-Test: True",
		"--body", req.Body,
		"--hide-all",
	}

	cmdStr := fmt.Sprintf("swaks %s", strings.Join(args, " "))

	cmd := exec.Command("swaks", args...)
	output, err := cmd.CombinedOutput()

	response := map[string]string{
		"status":      "sent",
		"sender_ip":   sender.IP,
		"used_helo":   helo,
		"message_id":  msgID, // (Optional) Return ID in response for debugging
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
