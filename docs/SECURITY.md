# Security Policy

## Security Features

KumoMTA UI implements robust security measures to protect your email infrastructure (v2.1+):

### 🔐 Authentication & Access

- **Two-Factor Authentication (2FA):** Mandatory 2FA is supported and recommended for all admin accounts. Uses standard TOTP (compatible with Google Authenticator, Authy, etc.).
- **Session Management:** Active sessions are tracked by IP and User Agent. You can remotely revoke sessions from the Security dashboard.
- **Secure Login Flow:** 2FA verification happens in a separate, secure step using temporary tokens.

### 🛡️ Infrastructure Protection

- **Security Audit Tool:** Built-in scanner (available in Webhooks menu) checks for:
  - World-readable database files.
  - Publicly exposed HTTP ports (bypassing Nginx).
  - Missing API keys.
- **Blacklist Monitoring:** Automatically scans your server IPs against major RBLs (Spamhaus, Barracuda) hourly.

### 💻 Data Protection

- **Password Hashing:** All passwords (admin, bounce accounts) are hashed using **bcrypt**.
- **Write-Only Secrets:** SMTP passwords and AI API keys are never returned in API responses.
- **Input Validation:** Strict sanitization on all system-level inputs (e.g., bounce usernames) to prevent shell injection.

### 🌐 Network Security

- **CORS Policy:** Strict Allow-Origin enforcement to prevent malicious cross-site requests.
- **Localhost Binding:** The backend API listens only on `127.0.0.1:9000` by default, forcing all traffic through the Nginx reverse proxy (SSL).

---

## Deployment Recommendations

### 1. Use HTTPS

Always deploy with HTTPS. The provided installer sets up Let's Encrypt automatically.

### 2. Firewall Configuration

Only expose necessary ports. Port `9000` should **NEVER** be open to the public internet.

```bash
# Correct Firewall Setup
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --permanent --remove-port=9000/tcp
firewall-cmd --reload
```

### 3. Regular Audits

Use the "Run Security Audit" button in the dashboard after any server configuration change to ensure no vulnerabilities were introduced.

---

## Reporting Security Issues

If you discover a security vulnerability, please do not open a public GitHub issue. Contact the maintainer directly. We commit to addressing critical security issues within 48 hours.
