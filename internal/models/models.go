package models

import "time"

// Represents global application settings.
type AppSettings struct {
	ID uint `gorm:"primaryKey"`

	MainHostname string
	MainServerIP string
	MailWizzIP   string // optional relay IP

	AIProvider string // "openai", "deepseek"
	AIAPIKey   string // encrypted or blank

	// Webhook Settings
	WebhookURL     string // Slack/Discord webhook URL
	WebhookEnabled bool
	BounceAlertPct float64 // Alert when bounce rate exceeds this %
}

// A domain managed by the system
type Domain struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex"`

	MailHost   string
	BounceHost string

	// DMARC Settings
	DMARCPolicy     string // none, quarantine, reject
	DMARCRua        string // Aggregate report email
	DMARCRuf        string // Forensic report email
	DMARCPercentage int    // 0-100

	Senders []Sender `gorm:"constraint:OnDelete:CASCADE"`
}

// AdminUser represents a panel admin account
type AdminUser struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string // bcrypt hash

	// 2FA Support
	TwoFactorSecret  string // TOTP secret (encrypted)
	TwoFactorEnabled bool

	// User Preferences
	Theme string // "dark", "light", "system"
}

// Auth Sessions for multi-device support
type AuthSession struct {
	ID        uint      `gorm:"primaryKey"`
	AdminID   uint      `gorm:"index"`
	Token     string    `gorm:"uniqueIndex"`
	ExpiresAt time.Time
	CreatedAt time.Time
	DeviceIP  string
	UserAgent string // Browser/device info
}

// BounceAccount represents a system user for handling bounced emails
type BounceAccount struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	Domain       string
	Notes        string
}

// A sender identity associated with a domain
type Sender struct {
	ID       uint `gorm:"primaryKey"`
	DomainID uint `gorm:"index"`

	LocalPart    string
	Email        string
	IP           string // specific IP for this sender
	SMTPPassword string
}

// Inventory of IPs available on the server
type SystemIP struct {
	ID        uint      `gorm:"primaryKey"`
	Value     string    `gorm:"uniqueIndex"` // IPv4 address
	Netmask   string    // e.g. /24
	Interface string    // e.g. eth0 (optional)
	CreatedAt time.Time
}

// EmailStats stores aggregated sending statistics
type EmailStats struct {
	ID        uint      `gorm:"primaryKey"`
	Domain    string    `gorm:"index"`
	Date      time.Time `gorm:"index"` // Date only (no time)
	Sent      int64
	Delivered int64
	Bounced   int64
	Deferred  int64
	UpdatedAt time.Time
}

// QueueMessage represents a message in the mail queue
type QueueMessage struct {
	ID          string    `json:"id"`
	Sender      string    `json:"sender"`
	Recipient   string    `json:"recipient"`
	Subject     string    `json:"subject"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"` // queued, deferred, etc.
	Attempts    int       `json:"attempts"`
	LastAttempt time.Time `json:"last_attempt"`
	NextRetry   time.Time `json:"next_retry"`
	ErrorMsg    string    `json:"error_msg"`
}

// WebhookLog stores webhook delivery history
type WebhookLog struct {
	ID        uint      `gorm:"primaryKey"`
	EventType string    // bounce_alert, daily_summary
	Payload   string    // JSON payload sent
	Status    int       // HTTP status code
	Response  string    // Response body
	CreatedAt time.Time
}
