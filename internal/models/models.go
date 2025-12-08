package models

import "time"

// Represents global application settings.
// Only one row is expected.
type AppSettings struct {
	ID uint `gorm:"primaryKey"`

	MainHostname string // e.g. mta.server.com
	MainServerIP string // e.g. 54.x.x.x
	MailWizzIP   string // optional relay IP (also used for generic relay IPs)

	AIProvider string // "openai", "deepseek", ""
	AIAPIKey   string // encrypted or blank
}

// A domain managed by the system (completely generic)
type Domain struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex"` // example: example.com

	MailHost   string // mail.example.com
	BounceHost string // bounce.example.com

	Senders []Sender `gorm:"constraint:OnDelete:CASCADE"`
}

// AdminUser represents a panel admin account for authentication.
type AdminUser struct {
	ID           uint      `gorm:"primaryKey"`
	Email        string    `gorm:"uniqueIndex"`
	PasswordHash string    // bcrypt hash
	APIToken     string    `gorm:"index"` // random token for API auth
	TokenExpiry  time.Time // Token expiration time
}

// BounceAccount represents a system user for handling bounced emails.
type BounceAccount struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"` // system username, IMAP login
	PasswordHash string // bcrypt hash
	Domain       string // optional, for reference
	Notes        string // optional description
}

// A sender identity associated with a domain
// (Generic: can be ANYTHING)
type Sender struct {
	ID       uint `gorm:"primaryKey"`
	DomainID uint `gorm:"index"`

	// Generic fields editable from UI
	LocalPart    string // "editor", "info", "support", etc.
	Email        string // auto = localpart@domain
	IP           string // chosen sending IP
	SMTPPassword string // editable
}
