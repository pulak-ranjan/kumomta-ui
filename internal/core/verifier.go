package core

import (
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// EmailVerificationResult holds the outcome of a check
type EmailVerificationResult struct {
	Email      string `json:"email"`
	IsValid    bool   `json:"is_valid"`
	IsCatchAll bool   `json:"is_catch_all"`
	Error      string `json:"error,omitempty"`
	RiskScore  int    `json:"risk_score"` // 0 = safe, 100 = invalid/risky
	Log        string `json:"log"`
}

// VerifierOptions configures the check
type VerifierOptions struct {
	SenderEmail string
	HeloHost    string
	SourceIPs   []string // List of local IPs to rotate
	ProxyURL    string   // Fallback proxy (SOCKS5/HTTP)
}

// VerifyEmail performs robust checks with Multi-IP and Proxy fallback
func VerifyEmail(email string, opts VerifierOptions) EmailVerificationResult {
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
	mxHost = strings.TrimSuffix(mxHost, ".") // Ensure no trailing dot
	res.Log = fmt.Sprintf("MX found: %s. ", mxHost)

	// 3. SMTP Handshake with Multi-IP Rotation

	// Prepare list of Dialers (Source IPs + Default + Proxy)
	// Strategy: Try IPs first. If connection fails/blocked, try Proxy.

	dialers := make([]func(network, addr string) (net.Conn, error), 0)

	// A. Add Source IPs
	for _, ip := range opts.SourceIPs {
		localIP := ip // capture closure
		dialers = append(dialers, func(network, addr string) (net.Conn, error) {
			localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
			if err != nil { return nil, err }
			d := net.Dialer{LocalAddr: localAddr, Timeout: 10 * time.Second}
			return d.Dial(network, addr)
		})
	}

	// B. Add Default Interface (if no source IPs or just as backup)
	if len(dialers) == 0 {
		dialers = append(dialers, func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 10*time.Second)
		})
	}

	// C. Add Proxy (Fallback)
	if opts.ProxyURL != "" {
		dialers = append(dialers, func(network, addr string) (net.Conn, error) {
			u, err := url.Parse(opts.ProxyURL)
			if err != nil { return nil, err }
			d, err := proxy.FromURL(u, proxy.Direct)
			if err != nil { return nil, err }
			return d.Dial(network, addr)
		})
	}

	// Try each dialer until one gives a definitive answer (or all fail)
	for i, dial := range dialers {
		res.Log += fmt.Sprintf("[Attempt %d] ", i+1)

		result := performSMTPCheck(dial, mxHost, email, opts)

		if result.Error == "" {
			// Success! Server accepted RCPT TO
			res.IsValid = true
			res.IsCatchAll = result.IsCatchAll
			res.RiskScore = 0
			if result.IsCatchAll {
				res.RiskScore = 50 // Catch-all means we can't be sure
				res.Log += "Catch-All Detected (Unknown)."
			} else {
				res.Log += "Success."
			}
			res.Error = ""
			return res
		}

		// Analyze Error
		// If 550 User Unknown -> Stop, we know it doesn't exist.
		// If Timeout/Connect Fail -> Continue to next IP/Proxy.
		if strings.Contains(result.Error, "550") || strings.Contains(result.Error, "User unknown") {
			res.IsValid = false
			res.IsCatchAll = result.IsCatchAll
			res.RiskScore = 100
			res.Error = result.Error
			res.Log += "Rejected (550)."
			return res
		}

		res.Log += fmt.Sprintf("Failed (%s). Retrying... ", result.Error)
	}

	// If all attempts failed (timeouts/blocks)
	res.IsValid = false
	res.RiskScore = 50 // Unknown/Grey
	res.Error = "All connection attempts failed"
	return res
}

type smtpCheckResult struct {
	IsCatchAll bool
	Error      string
}

func performSMTPCheck(dial func(network, addr string) (net.Conn, error), host, email string, opts VerifierOptions) smtpCheckResult {
	conn, err := dial("tcp", fmt.Sprintf("%s:25", host))
	if err != nil {
		return smtpCheckResult{Error: fmt.Sprintf("Connect error: %v", err)}
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return smtpCheckResult{Error: fmt.Sprintf("Client error: %v", err)}
	}
	defer client.Quit()

	helo := opts.HeloHost
	if helo == "" { helo = "check.kumomta.local" }

	if err := client.Hello(helo); err != nil {
		return smtpCheckResult{Error: fmt.Sprintf("HELO error: %v", err)}
	}

	sender := opts.SenderEmail
	if sender == "" { sender = fmt.Sprintf("verifier@%s", helo) }

	if err := client.Mail(sender); err != nil {
		return smtpCheckResult{Error: fmt.Sprintf("MAIL FROM error: %v", err)}
	}

	// 1. Catch-All Check (Reacher Backend Logic)
	// Try a random invalid email to see if server accepts everything
	randomLocal := fmt.Sprintf("random-%d", time.Now().UnixNano())
	domain := strings.Split(email, "@")[1]
	randomEmail := fmt.Sprintf("%s@%s", randomLocal, domain)

	// We ignore error here because we just want to know 250 vs 550
	err = client.Rcpt(randomEmail)
	if err == nil {
		// Server ACCEPTED a garbage email -> Catch-All detected
		// We stop here because verifying the real email provides no info
		return smtpCheckResult{IsCatchAll: true, Error: ""}
	}

	// If rejected (550), it's NOT a catch-all, so we can trust the next check.
	// Reset state? No, RCPT can be called multiple times in one session usually.
	// But some servers might be picky. Let's try proceeding.

	// 2. Real Email Check
	if err := client.Rcpt(email); err != nil {
		return smtpCheckResult{Error: fmt.Sprintf("RCPT TO error: %v", err)}
	}

	return smtpCheckResult{IsCatchAll: false, Error: ""}
}
