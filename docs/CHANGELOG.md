# Changelog

All notable changes to KumoMTA UI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
