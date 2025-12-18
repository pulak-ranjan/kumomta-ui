package models

import "time"

// Represents global application settings.
type AppSettings struct {
	ID uint `gorm:"primaryKey" json:"id"`

	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`
	MailWizzIP   string `json:"mailwizz_ip"` // optional relay IP

	// NEW: Listener Binding (e.g., "127.0.0.1:25" or "0.0.0.0:25")
	SMTPListenAddr string `json:"smtp_listen_addr"`

	AIProvider string `json:"ai_provider"` // "openai", "deepseek"
	AIAPIKey   string `json:"ai_api_key"`  // encrypted or blank

	// Webhook Settings
	WebhookURL     string  `json:"webhook_url"`
	WebhookEnabled bool    `json:"webhook_enabled"`
	BounceAlertPct float64 `json:"bounce_alert_pct"`
}

// A domain managed by the system
type Domain struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex" json:"name"`

	MailHost   string `json:"mail_host"`
	BounceHost string `json:"bounce_host"`

	// DMARC Settings
	DMARCPolicy     string `json:"dmarc_policy"`     // none, quarantine, reject
	DMARCRua        string `json:"dmarc_rua"`        // Aggregate report email
	DMARCRuf        string `json:"dmarc_ruf"`        // Forensic report email
	DMARCPercentage int    `json:"dmarc_percentage"` // 0-100

	Senders []Sender `gorm:"constraint:OnDelete:CASCADE" json:"senders"`
}

// AdminUser represents a panel admin account
type AdminUser struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Email        string `gorm:"uniqueIndex" json:"email"`
	PasswordHash string `json:"-"` // Never return hash

	// 2FA Support
	TwoFactorSecret  string `json:"-"`
	TwoFactorEnabled bool   `json:"has_2fa"`

	// User Preferences
	Theme string `json:"theme"`
}

// Auth Sessions for multi-device support
type AuthSession struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AdminID   uint      `gorm:"index" json:"admin_id"`
	Token     string    `gorm:"uniqueIndex" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	DeviceIP  string    `json:"device_ip"`
	UserAgent string    `json:"user_agent"`
}

// BounceAccount represents a system user for handling bounced emails
type BounceAccount struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex" json:"username"`
	PasswordHash string `json:"-"`
	Domain       string `json:"domain"`
	Notes        string `json:"notes"`
}

// A sender identity associated with a domain
type Sender struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	DomainID uint `gorm:"index" json:"domain_id"`
	Domain   Domain `json:"-" gorm:"foreignKey:DomainID"` // Relation for Warmup Logic

	LocalPart    string `json:"local_part"`
	Email        string `json:"email"`
	IP           string `json:"ip"` // specific IP for this sender
	SMTPPassword string `json:"smtp_password,omitempty"`
	
	BounceUsername string `json:"bounce_username"`

	// Warmup State (NEW)
	WarmupEnabled    bool      `json:"warmup_enabled"`
	WarmupPlan       string    `json:"warmup_plan"`        // "conservative", "standard"
	WarmupDay        int       `json:"warmup_day"`         // 1, 2, 3...
	WarmupLastUpdate time.Time `json:"warmup_last_update"` // Last time we bumped the day

	// Virtual field for DKIM check (computed at runtime)
	HasDKIM bool `gorm:"-" json:"has_dkim"` 
}

// Inventory of IPs available on the server
type SystemIP struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Value     string    `gorm:"uniqueIndex" json:"value"` // IPv4 address
	Netmask   string    `json:"netmask"`                  // e.g. /24
	Interface string    `json:"interface"`                // e.g. eth0 (optional)
	CreatedAt time.Time `json:"created_at"`

	// Virtual Field: True if this IP is actually configured on the OS
	IsActive  bool      `gorm:"-" json:"is_active"` 
}

// EmailStats stores aggregated sending statistics
type EmailStats struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"index" json:"domain"`
	Date      time.Time `gorm:"index" json:"date"` // Date only (no time)
	Sent      int64     `json:"sent"`
	Delivered int64     `json:"delivered"`
	Bounced   int64     `json:"bounced"`
	Deferred  int64     `json:"deferred"`
	UpdatedAt time.Time `json:"updated_at"`
}

// QueueMessage represents a message in the mail queue (RESTORED)
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
	ID        uint      `gorm:"primaryKey" json:"id"`
	EventType string    `json:"event_type"` // bounce_alert, daily_summary
	Payload   string    `json:"payload"`    // JSON payload sent
	Status    int       `json:"status"`     // HTTP status code
	Response  string    `json:"response"`   // Response body
	CreatedAt time.Time `json:"created_at"`
}

// APIKey for external applications (NEW)
type APIKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`                   // e.g. "EmailVerifier App"
	Key       string    `gorm:"uniqueIndex" json:"key"` // The secret token
	Scopes    string    `json:"scopes"`                 // e.g. "verify,relay"
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// ChatLog stores AI conversation history (NEW)
type ChatLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Role      string    `json:"role"` // "user", "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Campaign represents a bulk email job
type Campaign struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`          // HTML Content
	SenderID    uint      `json:"sender_id"`     // From which Sender identity
	Sender      Sender    `json:"-" gorm:"foreignKey:SenderID"`
	Status      string    `json:"status"`        // "draft", "sending", "completed", "failed"
	TotalSent   int       `json:"total_sent"`
	TotalFailed int       `json:"total_failed"`
	CreatedAt   time.Time `json:"created_at"`
	Recipients  []CampaignRecipient `json:"recipients,omitempty" gorm:"foreignKey:CampaignID"`
}

// CampaignRecipient tracks individual status in a campaign
type CampaignRecipient struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CampaignID uint      `gorm:"index" json:"campaign_id"`
	Email      string    `gorm:"index" json:"email"`
	Status     string    `json:"status"` // "pending", "sent", "failed"
	Error      string    `json:"error,omitempty"`
	SentAt     time.Time `json:"sent_at,omitempty"`
}
