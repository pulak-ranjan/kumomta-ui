# Changelog

All notable changes to KumoMTA UI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-12-09

### Added
- Initial production release
- Admin authentication with JWT-style tokens
- Token expiration (7-day validity, regenerated on login)
- Email and password validation on registration
- Domain management with CRUD operations
- Sender management per domain
- DKIM key generation (RSA 2048-bit)
- DNS record helpers (A, MX, SPF, DKIM TXT)
- Bounce account management with system user provisioning
- KumoMTA configuration generator:
  - `sources.toml`
  - `queues.toml`
  - `listener_domains.toml`
  - `dkim_data.toml`
  - `init.lua`
- Configuration validation and apply with auto-restart
- System log viewers (KumoMTA, Dovecot, Fail2Ban)
- Service status dashboard
- App settings management
- Security headers middleware
- Atomic file writes for config safety
- Auto-installer for Rocky Linux 9 with optional HTTPS

### Security
- Passwords hashed with bcrypt
- SMTP passwords write-only (never returned in API)
- AI API keys write-only
- Token expiration enforcement
- Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)

---

## [Unreleased]

### Planned
- Multi-admin support
- Password reset functionality
- Email sending statistics dashboard
- DMARC record generator
- Rate limiting middleware
- Audit logging
- Docker deployment option
