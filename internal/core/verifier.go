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

// Reacher-compatible Result Structure
type EmailVerificationResult struct {
	Input       string `json:"input"`
	IsReachable string `json:"is_reachable"` // "safe", "risky", "invalid", "unknown"

	Misc   MiscDetails   `json:"misc"`
	MX     MXDetails     `json:"mx"`
	SMTP   SMTPDetails   `json:"smtp"`
	Syntax SyntaxDetails `json:"syntax"`

	Error     string `json:"error,omitempty"`
	RiskScore int    `json:"risk_score"`
	Log       string `json:"log"`
}

type MiscDetails struct {
	IsDisposable   bool `json:"is_disposable"`
	IsRoleAccount  bool `json:"is_role_account"`
}

type MXDetails struct {
	AcceptsMail bool     `json:"accepts_mail"`
	Records     []string `json:"records"`
}

type SMTPDetails struct {
	CanConnect    bool `json:"can_connect_smtp"`
	IsCatchAll    bool `json:"is_catch_all"`
	IsDeliverable bool `json:"is_deliverable"`
}

type SyntaxDetails struct {
	Domain        string `json:"domain"`
	Username      string `json:"username"`
	IsValidSyntax bool   `json:"is_valid_syntax"`
}

// Common disposable domains (truncated list for MVP)
var disposableDomains = map[string]bool{
	"mailinator.com": true, "yopmail.com": true, "guerrillamail.com": true,
	"temp-mail.org": true, "10minutemail.com": true, "sharklasers.com": true,
}

// Common role accounts
var roleAccounts = map[string]bool{
	"admin": true, "support": true, "info": true, "sales": true, "contact": true,
	"webmaster": true, "postmaster": true, "hostmaster": true, "abuse": true,
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
	res := EmailVerificationResult{Input: email}

	// 1. Syntax Check
	parts := strings.Split(email, "@")
	if len(parts) != 2 || !strings.Contains(parts[1], ".") {
		res.IsReachable = "invalid"
		res.Syntax.IsValidSyntax = false
		res.Error = "Invalid syntax"
		return res
	}
	res.Syntax.IsValidSyntax = true
	res.Syntax.Username = parts[0]
	res.Syntax.Domain = parts[1]

	// Misc Checks
	res.Misc.IsDisposable = disposableDomains[parts[1]]
	res.Misc.IsRoleAccount = roleAccounts[parts[0]]

	// 2. MX Record Lookup
	mxs, err := net.LookupMX(res.Syntax.Domain)
	if err != nil || len(mxs) == 0 {
		res.IsReachable = "invalid"
		res.MX.AcceptsMail = false
		res.Error = "No MX records found"
		return res
	}

	res.MX.AcceptsMail = true
	for _, mx := range mxs {
		res.MX.Records = append(res.MX.Records, mx.Host)
	}

	mxHost := mxs[0].Host
	mxHost = strings.TrimSuffix(mxHost, ".") // Ensure no trailing dot

	// 3. SMTP Handshake with Multi-IP Rotation

	// Prepare list of Dialers (Source IPs + Default + Proxy)
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

	// B. Add Default Interface
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

	// Try each dialer
	for i, dial := range dialers {
		res.Log += fmt.Sprintf("[Attempt %d] ", i+1)

		result := performSMTPCheck(dial, mxHost, email, opts)

		// Fill SMTP details
		res.SMTP.CanConnect = (result.Error == "" || strings.Contains(result.Error, "550") || strings.Contains(result.Error, "RCPT"))
		res.SMTP.IsCatchAll = result.IsCatchAll

		if result.Error == "" {
			// Success
			if result.IsCatchAll {
				res.IsReachable = "risky" // Catch-all is always risky/unknown
				res.SMTP.IsDeliverable = true // Technically yes, but...
				res.RiskScore = 50
			} else {
				res.IsReachable = "safe"
				res.SMTP.IsDeliverable = true
				res.RiskScore = 0
			}
			return res
		}

		// Analyze Error
		if strings.Contains(result.Error, "550") || strings.Contains(result.Error, "User unknown") {
			res.IsReachable = "invalid"
			res.SMTP.IsDeliverable = false
			res.RiskScore = 100
			return res
		}

		res.Log += fmt.Sprintf("Failed (%s). Retrying... ", result.Error)
	}

	// If all attempts failed
	res.IsReachable = "unknown"
	res.RiskScore = 50
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
