# KumoMTA UI

A modern, production-grade control panel for managing KumoMTA, domains, senders, DKIM keys, relay IPs, bounce accounts, and configuration files — with a clean React frontend and a robust Go backend.

**Project by:** Pulak Ranjan  
**Developed with the help of:** ChatGPT and Gemini

---

## ✨ Features

### 🔧 Backend (Go)

- Fully modular backend written in Go 1.22
- Secure admin authentication (JWT)
- Full CRUD:
  - Domains
  - Senders (local parts, SMTP IPs, SMTP password write-only)
  - Bounce mailboxes (bcrypt hashed passwords)
- System bounce account provisioning (Linux user creation, Maildir setup)
- App settings (hostname, server IP, relay IPs, AI provider keys)
- Automatic DKIM RSA key generation (2048-bit)
- DNS record exporter for DKIM + A + MX + SPF
- Config generator for:
  - `sources.toml`
  - `queues.toml`
  - `listener_domains.toml`
  - `dkim_data.toml`
  - `init.lua`
- Config apply (validate + restart KumoMTA)
- System-level log access:
  - journalctl for KumoMTA
  - journalctl for Dovecot
  - journalctl for Fail2Ban
- Local or Docker builds supported
- Environment-based DB path (`DB_DIR`)

### 🎨 Frontend (React + Tailwind)

- Clean dashboard UI
- First-time setup screen (admin creation)
- Real-time DNS helpers for each domain
- DKIM generator UI with:
  - Copy Name
  - Copy Value
  - Copy Full TXT
- Domain and sender management
- Bounce management UI
- Log viewers for Kumo, Dovecot, Fail2Ban
- Status API viewer
- Auto-detected Panel URL + API URL

---

## 🛠️ Requirements

### Server

- Rocky Linux 9 (recommended)
- nginx (optional for proxy + HTTPS)
- certbot (optional for Let's Encrypt SSL)
- firewalld
- systemd
- KumoMTA installed at `/opt/kumomta`

### Client

- Any modern browser
- HTTPS recommended

---

## 🚀 Installation Guide (Production – Rocky Linux 9)

You will find an auto-installer in:

```
scripts/install-kumomta-ui-rocky9.sh
```

### 1. Clone the repository

```bash
sudo mkdir -p /opt/kumomta-ui
sudo git clone https://github.com/pulak-ranjan/kumomta-ui.git /opt/kumomta-ui
cd /opt/kumomta-ui
```

### 2. Run the installer

```bash
sudo bash scripts/install-kumomta-ui-rocky9.sh
```

During installation, it will ask:

- System hostname (optional)
- Panel domain for HTTPS (optional)
- Email for Let's Encrypt (required if domain provided)

**If domain is provided:**
- Nginx reverse proxy is created automatically
- Let's Encrypt SSL is issued via certbot
- Firewall ports 80 and 443 opened

**If no domain is provided:**
- Panel is available at `http://SERVER-IP:9000`
- Firewall port 9000 opened

The installer prints your final access URL, e.g.:

```
Panel URL: https://mta.example.com/
API URL:   https://mta.example.com/api
```

---

## 🔧 Development Build Instructions

### Backend (Go)

```bash
go mod tidy
go run ./cmd/server
```

Local DB:

```bash
export DB_DIR=./data
go run ./cmd/server
```

Build binary:

```bash
go build -o kumomta-ui-server ./cmd/server
```

### Frontend (React)

Install dependencies:

```bash
cd web
npm install
```

Run in dev mode:

```bash
npm run dev
```

Build static production files:

```bash
npm run build
```

Files appear in:

```
web/dist/
```

These will be served by Nginx or any static file hosting method you prefer. In production, the backend serves only API; Nginx proxies frontend → backend.

---

## 🔐 Security Features

- Admin password hashed using bcrypt
- Bounce mailbox passwords hashed
- SMTP password for senders is write-only (never returned to client)
- AI API keys stored encrypted or not returned in API
- Backend listens on local port when HTTPS domain is configured
- Nginx terminates TLS with automatic renewals

---

## 📡 API Structure (Quick Overview)

### Auth

```
POST /api/auth/register
POST /api/auth/login
GET  /api/auth/me
```

### Settings

```
GET  /api/settings
POST /api/settings
```

### Domains & Senders

```
GET    /api/domains
POST   /api/domains
GET    /api/domains/{id}
DELETE /api/domains/{id}

GET    /api/domains/{id}/senders
POST   /api/domains/{id}/senders
DELETE /api/senders/{id}
```

### DKIM

```
POST /api/dkim/generate
GET  /api/dkim/records
```

### Bounce

```
GET    /api/bounces
POST   /api/bounces
DELETE /api/bounces/{id}
POST   /api/bounces/apply
```

### Logs / Status

```
GET /api/status
GET /api/logs/kumomta
GET /api/logs/dovecot
GET /api/logs/fail2ban
```

### Config

```
GET  /api/config/preview
POST /api/config/apply
```

---

## 📦 Build Versioning

Follow this structure:

```
v1.0.0  – First production-ready release
v1.1.0  – DKIM improvements, UI enhancements
v1.2.0  – Auto HTTPS installer, bounce hashing
v1.3.0  – AI assistant integration (future)
```

Semantic Versioning:

```
MAJOR.MINOR.PATCH
```

---

## ✍️ Credits

This project was created by:

**Pulak Ranjan**

with development assistance from:

- **ChatGPT** (OpenAI)
- **Gemini** (Google)

---

## ❤️ Contribution

PRs are welcome!

Open issues if you want new features or need improvements.

---

## 📜 License

[Apache License 2.0](LICENSE)
