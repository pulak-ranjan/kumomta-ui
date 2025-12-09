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
}

// A domain managed by the system
type Domain struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex"`

	MailHost   string
	BounceHost string

	Senders []Sender `gorm:"constraint:OnDelete:CASCADE"`
}

// AdminUser represents a panel admin account
type AdminUser struct {
	ID           uint      `gorm:"primaryKey"`
	Email        string    `gorm:"uniqueIndex"`
	PasswordHash string    // bcrypt hash
	APIToken     string    `gorm:"index"`
	TokenExpiry  time.Time
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

//Inventory of IPs available on the server
type SystemIP struct {
	ID        uint      `gorm:"primaryKey"`
	Value     string    `gorm:"uniqueIndex"` // IPv4 address
	Netmask   string    // e.g. /24
	Interface string    // e.g. eth0 (optional)
	CreatedAt time.Time
}
