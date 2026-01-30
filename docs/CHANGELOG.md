# Changelog

All notable changes to KumoMTA UI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.2.0] - 2026-01-30

### Deliverability Engine
- **Envelope-From Separation:** Automatic `localpart@localpart.domain.com` envelope rewriting for per-sender bounce isolation.
- **ISP Traffic Shaping:** Built-in rate limits for Gmail (50/h), Microsoft (50/h), Yahoo (100/h) with per-ISP connection limits. Matching uses MX hostnames (site_name) for correct ISP identification.
- **Connection Limits:** Default `max_connection_rate`, `max_deliveries_per_connection`, and `connection_limit` for all outbound traffic.
- **Listener Subdomain Auto-Generation:** `listener_domains.toml` now includes sender subdomains (e.g., `a1.domain.com`) for envelope-from bounce handling.

### SMTP Authentication
- **Fixed 3-Parameter Auth:** `smtp_server_auth_plain` now correctly uses `(authzid, authcid, password)` — previously only used 2 parameters, causing all authentication to fail.
- **Authenticated Relay:** Authenticated users are now automatically allowed to relay via `conn_meta:get_meta('authz_id')` check in `get_listener_domain`.
- **TLS Configuration:** Added TLS certificate/key settings for SMTP listeners (ports 25, 587, 465). Port 465 starts with implicit TLS.

### DKIM & Headers
- **List-Unsubscribe-Post:** Added to DKIM signed headers for Gmail one-click unsubscribe compliance.
- **Header Scrubbing:** Removes `User-Agent`, `X-Mailer`, `X-Originating-IP`, and internal X-headers to prevent MTA fingerprinting.
- **Stealth Received Header:** Injects Postfix-style Received header to avoid KumoMTA identification.

### Queue & Retry
- **Retry Interval:** Changed from 1 minute to 5 minutes to avoid aggressive retry storms.

### Frontend
- **SMTP Info Modal:** Fixed to show actual server hostname and sender credentials instead of hardcoded dummy values.
- **TLS Settings UI:** Added TLS certificate and key path configuration in Settings page.

### Documentation
- **DNS-SETUP.md:** Complete DNS configuration guide including sender subdomain records.
- **DELIVERABILITY.md:** Full deliverability guide covering envelope separation, ISP shaping, bounce handling, warmup, and SMTP client setup.

---

## [2.1.0] - 2025-12-10

### 🚀 Major UI Overhaul
- **New Design System:** Completely rewritten frontend using Card-based layouts and Glassmorphism.
- **Icons:** Replaced all emojis with professional `lucide-react` SVG icons.
- **Theming:** Added robust Dark/Light mode support with system preference synchronization.
- **Mobile Responsive:** Added a collapsible sidebar and hamburger menu for mobile management.
- **Terminal Logs:** New "hacker-style" log viewer for KumoMTA/Dovecot/Fail2Ban logs.

### 🛡️ Security
- **Two-Factor Authentication (2FA):** Added TOTP support (Google Auth/Authy) for admin login.
- **Email Verification:** Enhanced input validation for email formats during registration and sender creation.
- **Security Audit Tool:** Automated scanner for dangerous file permissions and exposed ports.
- **CORS Fix:** Hardened API security to prevent unauthorized cross-origin requests.
- **Input Sanitization:** Enhanced validation for bounce account usernames to prevent shell injection.

### ⚡ Automation & Webhooks
- **Background Scheduler:** Added an internal scheduler for recurring tasks.
- **Webhook Integration:** Native support for **Discord** and **Slack** notifications.
- **Audit Logging:** Actions (Create/Delete Domain, etc.) now trigger webhook alerts.
- **Blacklist Monitor:** Hourly checks against Spamhaus and Barracuda RBLs.
- **Daily Reports:** Automated 24h traffic summary sent via webhook.

### 🔧 Fixes
- Fixed `AuthContext` to correctly handle 2FA challenges during login.
- Fixed API listening address to bind only to `127.0.0.1` (localhost) for Nginx security.
- Improved error handling in the "Config Apply" workflow.

---

## [1.0.0] - 2025-12-09

### Added
- Initial production release.
- Admin authentication with JWT-style tokens.
- Domain management with CRUD operations.
- Sender management per domain.
- DKIM key generation (RSA 2048-bit).
- KumoMTA configuration generator (`sources.toml`, `init.lua`, etc.).
- System log viewers.
- Auto-installer for Rocky Linux 9.
