package core

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// EmailVerificationResult holds the outcome of a check
type EmailVerificationResult struct {
	Email     string `json:"email"`
	IsValid   bool   `json:"is_valid"`
	Error     string `json:"error,omitempty"`
	RiskScore int    `json:"risk_score"` // 0 = safe, 100 = invalid/risky
	Log       string `json:"log"`
}

// VerifyEmail performs Syntax, DNS MX, and SMTP RCPT checks
func VerifyEmail(email string, senderEmail string) EmailVerificationResult {
	res := EmailVerificationResult{Email: email}

	// 1. Syntax Check
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		res.IsValid = false
		res.RiskScore = 100
		res.Error = "Invalid syntax"
		res.Log = "Syntax check failed"
		return res
	}

	parts := strings.Split(email, "@")
	domain := parts[1]

	// 2. MX Record Lookup
	mxs, err := net.LookupMX(domain)
	if err != nil || len(mxs) == 0 {
		res.IsValid = false
		res.RiskScore = 90
		res.Error = "No MX records found"
		res.Log = fmt.Sprintf("MX lookup failed for %s", domain)
		return res
	}

	mxHost := mxs[0].Host
	// Ensure no trailing dot for strict SMTP clients
	mxHost = strings.TrimSuffix(mxHost, ".")

	res.Log = fmt.Sprintf("MX found: %s. ", mxHost)

	// 3. SMTP Handshake (The "Reacher" part)
	// We connect, say HELO, MAIL FROM, and RCPT TO.
	// If RCPT TO is accepted (250), the email likely exists.
	// NOTE: Some servers accept everything (catch-all). Some block dynamic IPs.

	client, err := smtp.Dial(fmt.Sprintf("%s:25", mxHost))
	if err != nil {
		res.IsValid = false // Maybe transient? But for now, mark invalid.
		res.RiskScore = 50 // Could be network issue
		res.Error = fmt.Sprintf("SMTP Connect failed: %v", err)
		res.Log += "SMTP Connect failed."
		return res
	}
	defer client.Quit()

	if err := client.Hello("check.kumomta.local"); err != nil {
		res.Error = "HELO failed"
		res.RiskScore = 20
		return res
	}

	// Use a dummy sender or the provided one
	if senderEmail == "" { senderEmail = "verifier@kumomta.local" }

	if err := client.Mail(senderEmail); err != nil {
		res.Error = fmt.Sprintf("MAIL FROM failed: %v", err)
		res.RiskScore = 30
		return res
	}

	if err := client.Rcpt(email); err != nil {
		// 550 usually means user unknown
		res.IsValid = false
		res.RiskScore = 100
		res.Error = fmt.Sprintf("RCPT TO failed: %v", err)
		res.Log += "Recipient rejected."
		return res
	}

	// If we got here, the server accepted the recipient
	res.IsValid = true
	res.RiskScore = 0
	res.Log += "SMTP verification successful."

	return res
}
