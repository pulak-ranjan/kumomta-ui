package models

// Represents global application settings.
// Only one row is expected.
type AppSettings struct {
	ID uint `gorm:"primaryKey"`

	MainHostname string // e.g. mta.server.com
	MainServerIP string // e.g. 54.x.x.x
	MailWizzIP   string // optional relay IP

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

// A sender identity associated with a domain
// (Generic: can be editor, info, support, billing, ANYTHING)
type Sender struct {
	ID       uint   `gorm:"primaryKey"`
	DomainID uint   `gorm:"index"`

	// Generic fields editable from UI
	LocalPart    string // "editor", "info", "support", etc.
	Email        string // auto = localpart@domain
	IP           string // chosen sending IP
	SMTPPassword string // editable
}
