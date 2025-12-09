# KumoMTA UI API Documentation

**Base URL:** `http://your-server:9000/api` or `https://your-domain/api`

## 🔐 Authentication

All protected endpoints require a Bearer token in the `Authorization` header:
`Authorization: Bearer <token>`

### Public Endpoints

#### Register Admin
Create the first admin account (only allowed if no admin exists).
- **POST** `/auth/register`
- **Body:** `{ "email": "admin@example.com", "password": "StrongPassword1!" }`

#### Login
Authenticate and receive a session token.
- **POST** `/auth/login`
- **Body:** `{ "email": "admin@example.com", "password": "..." }`
- **Response (Success):** `{ "token": "...", "email": "..." }`
- **Response (2FA Required):** `{ "requires_2fa": true, "temp_token": "..." }`

#### Verify 2FA (Login Step 2)
Complete login if 2FA is enabled.
- **POST** `/auth/verify-2fa`
- **Header:** `X-Temp-Token: <temp_token_from_login>`
- **Body:** `{ "code": "123456" }`
- **Response:** `{ "token": "...", "email": "..." }`

---

### Protected Profile & Security

#### Get Current User
- **GET** `/auth/me`

#### Logout
Invalidate the current session token.
- **POST** `/auth/logout`

#### Setup 2FA
Initialize TOTP setup (returns QR code URI).
- **POST** `/auth/setup-2fa`
- **Body:** `{ "password": "current_password" }`
- **Response:** `{ "secret": "...", "uri": "otpauth://..." }`

#### Enable 2FA
Confirm and activate 2FA using a code.
- **POST** `/auth/enable-2fa`
- **Body:** `{ "code": "123456" }`

#### Disable 2FA
Turn off 2FA authentication.
- **POST** `/auth/disable-2fa`
- **Body:** `{ "password": "...", "code": "123456" }`

#### Update Theme
Set user UI preference.
- **POST** `/auth/theme`
- **Body:** `{ "theme": "dark" }` (or `light`, `system`)

#### List Sessions
View active login sessions for the user.
- **GET** `/auth/sessions`

---

## 📧 Domains & Senders

#### List Domains
Get all configured domains and their senders.
- **GET** `/domains`

#### Create Domain
- **POST** `/domains`
- **Body:** `{ "name": "example.com", "mail_host": "mail.example.com", "bounce_host": "bounce.example.com" }`

#### Get Domain
- **GET** `/domains/{id}`

#### Update Domain
- **PUT** `/domains/{id}`
- **Body:** `{ "dmarc_policy": "reject", ... }`

#### Delete Domain
Deletes domain and all associated senders.
- **DELETE** `/domains/{id}`

---

#### List Senders
- **GET** `/domains/{domainID}/senders`

#### Create Sender
- **POST** `/domains/{domainID}/senders`
- **Body:** `{ "local_part": "newsletter", "ip": "1.2.3.4", "smtp_password": "..." }`

#### Update Sender
- **PUT** `/senders/{id}`

#### Delete Sender
- **DELETE** `/senders/{id}`

#### Auto-Setup Sender
Automatically generate DKIM keys and create a system bounce account for a sender.
- **POST** `/domains/{domainID}/senders/{id}/setup`

---

## 📬 Bounce Accounts

#### List Bounce Accounts
- **GET** `/bounces`

#### Create/Update Bounce Account
Creates DB entry AND system user (useradd).
- **POST** `/bounces`
- **Body:** `{ "username": "b-news", "domain": "example.com", "password": "..." }`

#### Delete Bounce Account
- **DELETE** `/bounces/{id}`

#### Apply System State
Ensure all DB bounce accounts exist as Linux users.
- **POST** `/bounces/apply`

---

## 🛡️ DNS & Authentication (DKIM/DMARC)

#### List DKIM Keys
View all active DKIM public keys and selectors.
- **GET** `/dkim/records`

#### Generate DKIM
Generate RSA-2048 keys for a domain or specific user.
- **POST** `/dkim/generate`
- **Body:** `{ "domain": "example.com", "local_part": "optional_selector" }`

#### Get DMARC Record
- **GET** `/dmarc/{domainID}`

#### Set DMARC Policy
Generate and save DMARC settings.
- **POST** `/dmarc/{domainID}`
- **Body:** `{ "policy": "quarantine", "percentage": 100, "rua": "..." }`

#### Get All DNS
Preview A, MX, SPF, DMARC, and DKIM records for a domain.
- **GET** `/dns/{domainID}`

---

## 🔔 Webhooks & Automation

#### Get Webhook Settings
- **GET** `/webhooks/settings`

#### Update Webhook Settings
Configure Slack/Discord integration.
- **POST** `/webhooks/settings`
- **Body:** `{ "webhook_url": "...", "webhook_enabled": true, "bounce_alert_pct": 5.0 }`

#### Test Webhook
Send a test payload.
- **POST** `/webhooks/test`
- **Body:** `{ "webhook_url": "..." }`

#### Webhook Logs
View recent webhook dispatch history.
- **GET** `/webhooks/logs`

#### Manual Trigger: Check Bounces
Analyze bounce rates immediately and alert if high.
- **POST** `/webhooks/check-bounces`

---

## 🖥️ System & Networking

#### Dashboard Stats
Get CPU, RAM, Disk, and Service status.
- **GET** `/dashboard/stats`

#### List System IPs
- **GET** `/system/ips`

#### Add IP
- **POST** `/system/ips`
- **Body:** `{ "value": "1.2.3.4", "interface": "eth0" }`

#### Bulk Add IPs
- **POST** `/system/ips/bulk`
- **Body:** `{ "ips": ["1.2.3.4", "5.6.7.8"] }`

#### Add IPs by CIDR
- **POST** `/system/ips/cidr`
- **Body:** `{ "cidr": "192.168.1.0/24" }`

#### Auto-Detect IPs
Scan network interfaces for available IPv4 addresses.
- **POST** `/system/ips/detect`

#### Manual Trigger: Check Blacklists
Scan system IPs against RBLs (Spamhaus, etc.) and alert via Webhook.
- **POST** `/system/check-blacklist`

#### Manual Trigger: Security Audit
Scan for file permission issues and open ports.
- **POST** `/system/check-security`

#### AI Analysis
Analyze logs or health data using OpenAI/DeepSeek.
- **POST** `/system/ai-analyze`
- **Body:** `{ "type": "logs" }` (or `"health"`)

---

## ⚙️ Configuration & Queue

#### Preview Config
Generate KumoMTA config files (Lua/TOML) in memory.
- **GET** `/config/preview`

#### Apply Config
Write configs to disk (`/opt/kumomta/etc/policy`) and restart service.
- **POST** `/config/apply`

#### View Queue
- **GET** `/queue`
- **Query:** `?limit=100`

#### Queue Stats
- **GET** `/queue/stats`

#### Delete Message
- **DELETE** `/queue/{id}`

#### Flush Queue
Force retry of deferred messages.
- **POST** `/queue/flush`

---

## 📝 Logs

#### Service Logs
View tail of system logs via `journalctl`.
- **GET** `/logs/kumomta?lines=100`
- **GET** `/logs/dovecot?lines=100`
- **GET** `/logs/fail2ban?lines=100`

---

## 📥 Import

#### Bulk Import
Import domains and senders from CSV.
- **POST** `/import/csv`
- **Form Data:** `file` (CSV file with headers: `domain, localpart, ip, password`)
