# Security Policy

## Security Features

KumoMTA UI implements several security measures to protect your email infrastructure:

### Authentication & Authorization

- **Password Hashing**: All passwords (admin, bounce accounts) are hashed using bcrypt with default cost factor
- **Token-Based Auth**: API uses Bearer tokens instead of session cookies
- **Token Expiration**: Tokens expire after 7 days and are regenerated on each login
- **Single Admin**: Only one admin account can be registered (first-time setup only)

### Data Protection

- **Write-Only Secrets**: SMTP passwords and AI API keys are never returned in API responses
- **Input Validation**: Email format and password strength validation on registration
- **SQL Injection Prevention**: GORM ORM with parameterized queries

### HTTP Security Headers

The following headers are set on all responses:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
```

### File System Security

- **Atomic Writes**: Configuration files are written atomically to prevent corruption
- **Permission Control**: DKIM private keys are stored with 0600 permissions
- **Validation Before Apply**: KumoMTA configs are validated before service restart

---

## Deployment Recommendations

### 1. Use HTTPS

Always deploy with HTTPS in production. The installer supports automatic Let's Encrypt setup:

```bash
sudo bash scripts/install-kumomta-ui-rocky9.sh
# Provide your domain when prompted
```

### 2. Firewall Configuration

Only expose necessary ports:

```bash
# With HTTPS (recommended)
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https

# Without HTTPS (development only)
firewall-cmd --permanent --add-port=9000/tcp

firewall-cmd --reload
```

### 3. Run Behind Reverse Proxy

In production, the backend should only listen on localhost with nginx handling TLS:

```nginx
server {
    listen 443 ssl;
    server_name mta.example.com;
    
    ssl_certificate /etc/letsencrypt/live/mta.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mta.example.com/privkey.pem;
    
    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4. Database Security

The SQLite database contains sensitive data:

```bash
# Secure the database directory
chmod 700 /var/lib/kumomta-ui
chmod 600 /var/lib/kumomta-ui/panel.db
```

### 5. Regular Backups

Back up the database regularly:

```bash
# Backup
cp /var/lib/kumomta-ui/panel.db /backup/panel-$(date +%Y%m%d).db

# Also backup DKIM keys
tar -czf /backup/dkim-$(date +%Y%m%d).tar.gz /opt/kumomta/etc/dkim/
```

---

## Reporting Security Issues

If you discover a security vulnerability, please **do not** open a public GitHub issue.

Instead, please contact the maintainer directly with:

1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

We will respond within 48 hours and work on a fix.

---

## Security Checklist

Before deploying to production, verify:

- [ ] HTTPS is enabled
- [ ] Strong admin password is set
- [ ] Firewall is configured
- [ ] Database file permissions are restricted
- [ ] Backup strategy is in place
- [ ] Server is kept updated

---

## Known Limitations

1. **Single Admin**: Currently supports only one admin user
2. **No Rate Limiting**: API does not implement rate limiting (rely on nginx/firewall)
3. **No Audit Log**: User actions are not logged (planned for future release)
4. **Session Management**: No way to invalidate all tokens (changing password doesn't revoke existing tokens)

These limitations are planned to be addressed in future releases.
