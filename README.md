# KumoMTA UI

![Version](https://img.shields.io/badge/version-2.1.0-blue.svg) ![License](https://img.shields.io/badge/license-Apache%202.0-green.svg) ![Status](https://img.shields.io/badge/status-production--ready-purple.svg)

A modern, mobile-responsive control panel for KumoMTA. Manage domains, monitor queues, rotate IPs, and secure your infrastructure with a beautiful React frontend and a robust Go backend.

---

## ✨ Key Features

### 🎨 Modern UI (v2.1)
- **Responsive Design:** Fully mobile-friendly layout with collapsible sidebars.
- **Theming:** Seamless Dark/Light mode switching (system sync).
- **Visuals:** Professional styling with Lucide icons and glass-morphism effects.
- **Terminal Viewer:** Real-time log streaming with a code-editor feel.

### 🛡️ Advanced Security
- **Two-Factor Authentication (2FA):** TOTP support (Google Authenticator/Authy).
- **Security Audits:** Automated scans for file permissions and exposed ports.
- **Blacklist Monitoring:** Hourly checks against Spamhaus and Barracuda RBLs.
- **Audit Logging:** Tracks every administrative action (Create/Update/Delete).

### ⚙️ Core Management
- **Domain & Sender CRUD:** Manage identities with DKIM auto-generation.
- **IP Inventory:** Track and rotate server IPs easily.
- **Queue Management:** Flush queues, delete messages, and view granular status.
- **Config Generator:** Auto-generate `init.lua`, `sources.toml`, and more.

### 🔔 Automation & Alerts
- **Webhooks:** Native integration with **Slack** and **Discord**.
- **Daily Reports:** Automated traffic summaries sent every 24 hours.
- **Bounce Alerts:** Real-time notifications when bounce rates exceed thresholds.

---

## 🛠️ Prerequisites

Before installing, ensure your server meets these requirements:

- **OS:** Rocky Linux 9 (Recommended) or AlmaLinux 9.
- **Access:** Root (`sudo -i`) privileges.
- **KumoMTA:** Must be installed (`/opt/kumomta`).
- **Dependencies:**
  - `git`
  - `curl`
  - `nginx` (for reverse proxy)
  - `certbot` (for SSL)

---

## 🚀 Installation Guide

### Option A: Auto-Installer (Recommended)

This script installs Go, Node.js, Nginx, SSL (Certbot), and sets up the systemd service automatically.

```bash
# 1. Update your system
sudo dnf update -y

# 2. Install Git
sudo dnf install -y git

# 3. Clone the repository
sudo mkdir -p /opt/kumomta-ui
sudo git clone https://github.com/pulak-ranjan/kumomta-ui.git /opt/kumomta-ui
cd /opt/kumomta-ui

# 4. Run the installer
sudo bash scripts/install-kumomta-ui-rocky9.sh
```

### Option B: Manual Build (Step-by-Step)

If you prefer to configure the environment yourself or are developing locally:

#### 1. Backend Setup

```bash
# Install Go
sudo dnf install -y golang

# Create DB Directory
export DB_DIR=/var/lib/kumomta-ui
sudo mkdir -p $DB_DIR
sudo chmod 700 $DB_DIR

# Build Binary
cd /opt/kumomta-ui
go mod tidy
go build -o kumomta-ui-server ./cmd/server
```

#### 2. Frontend Setup

```bash
# Install Node.js 20
sudo dnf module install -y nodejs:20

# Build Static Files
cd web
npm install
npm run build
# The build will be in ./dist
```

#### 3. Run Service

```bash
# Set DB path and run
export DB_DIR=/var/lib/kumomta-ui
./kumomta-ui-server
```

---

## 🔒 Security Best Practices

1. **Enable 2FA:** Immediately after logging in, go to the Security page and setup Two-Factor Authentication.

2. **Configure Webhooks:** Go to Webhooks and add a Discord/Slack URL to receive security alerts.

3. **Run Audit:** Click "Run Security Audit" on the Webhooks page to verify file permissions.

4. **Firewall:** Ensure port 9000 is NOT exposed to the public internet. Use Nginx as a reverse proxy (handled by the installer).

---

## 🤝 Contributing

We welcome contributions! Please see CONTRIBUTING.md for details.

1. Fork the repo.
2. Create a feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes.
4. Open a Pull Request.

---

## 📜 License

Distributed under the Apache 2.0 License. See [LICENSE](https://github.com/pulak-ranjan/kumomta-ui/blob/main/LICENSE) for more information.
