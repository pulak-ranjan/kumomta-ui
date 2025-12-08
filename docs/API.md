# KumoMTA UI API Documentation

Base URL: `http://your-server:9000/api` or `https://your-domain/api`

## Authentication

All protected endpoints require a Bearer token in the Authorization header:

```
Authorization: Bearer <token>
```

Tokens are valid for **7 days** and regenerated on each login.

---

## Public Endpoints

### GET /api/status

Check service status (no authentication required).

**Response:**
```json
{
  "api": "ok",
  "kumomta": "active",
  "dovecot": "active",
  "fail2ban": "active"
}
```

---

## Authentication

### POST /api/auth/register

Create the first admin user. Only works when no admin exists.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "SecurePass123"
}
```

**Validation:**
- Email must be valid format
- Password must be 8+ characters with at least 1 letter and 1 number

**Response (200):**
```json
{
  "token": "abc123...",
  "email": "admin@example.com"
}
```

**Error (403):** Admin already exists

---

### POST /api/auth/login

Authenticate and get a new token.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "SecurePass123"
}
```

**Response (200):**
```json
{
  "token": "xyz789...",
  "email": "admin@example.com"
}
```

**Error (401):** Invalid credentials

---

### GET /api/auth/me

Get current user info. **Requires authentication.**

**Response:**
```json
{
  "email": "admin@example.com"
}
```

---

## Settings

### GET /api/settings

Get application settings. **Requires authentication.**

**Response:**
```json
{
  "main_hostname": "mta.example.com",
  "main_server_ip": "1.2.3.4",
  "relay_ips": "127.0.0.1,10.0.0.5",
  "ai_provider": "openai"
}
```

Note: `ai_api_key` is never returned (write-only).

---

### POST /api/settings

Save application settings. **Requires authentication.**

**Request:**
```json
{
  "main_hostname": "mta.example.com",
  "main_server_ip": "1.2.3.4",
  "relay_ips": "127.0.0.1,10.0.0.5",
  "ai_provider": "openai",
  "ai_api_key": "sk-..."
}
```

**Response:**
```json
{
  "status": "ok"
}
```

---

## Domains

### GET /api/domains

List all domains with their senders. **Requires authentication.**

**Response:**
```json
[
  {
    "id": 1,
    "name": "example.com",
    "mail_host": "mail.example.com",
    "bounce_host": "bounce.example.com",
    "senders": [
      {
        "id": 1,
        "domain_id": 1,
        "local_part": "info",
        "email": "info@example.com",
        "ip": "1.2.3.4"
      }
    ]
  }
]
```

Note: `smtp_password` is never returned (write-only).

---

### GET /api/domains/{id}

Get a single domain by ID. **Requires authentication.**

**Response:** Same structure as single item in list.

---

### POST /api/domains

Create or update a domain. **Requires authentication.**

**Create (id = 0 or omitted):**
```json
{
  "name": "example.com",
  "mail_host": "mail.example.com",
  "bounce_host": "bounce.example.com"
}
```

**Update (id provided):**
```json
{
  "id": 1,
  "name": "example.com",
  "mail_host": "mail.example.com",
  "bounce_host": "bounce.example.com"
}
```

---

### DELETE /api/domains/{id}

Delete a domain and all its senders. **Requires authentication.**

**Response:**
```json
{
  "status": "deleted"
}
```

---

## Senders

### GET /api/domains/{domainId}/senders

List senders for a domain. **Requires authentication.**

---

### POST /api/domains/{domainId}/senders

Create or update a sender. **Requires authentication.**

**Request:**
```json
{
  "id": 0,
  "local_part": "info",
  "email": "info@example.com",
  "ip": "1.2.3.4",
  "smtp_password": "secret123"
}
```

---

### DELETE /api/senders/{id}

Delete a sender by ID. **Requires authentication.**

---

## DKIM

### GET /api/dkim/records

List all DKIM DNS records. **Requires authentication.**

**Response:**
```json
[
  {
    "domain": "example.com",
    "selector": "info",
    "dns_name": "info._domainkey.example.com",
    "dns_value": "v=DKIM1; k=rsa; p=MIIBIjAN..."
  }
]
```

---

### POST /api/dkim/generate

Generate DKIM keys. **Requires authentication.**

**For specific sender:**
```json
{
  "domain": "example.com",
  "local_part": "info"
}
```

**For all senders in domain:**
```json
{
  "domain": "example.com"
}
```

**Response:**
```json
{
  "status": "ok",
  "domain": "example.com",
  "message": "dkim keys generated for all senders"
}
```

---

## Bounce Accounts

### GET /api/bounces

List all bounce accounts. **Requires authentication.**

**Response:**
```json
[
  {
    "id": 1,
    "username": "bounce1",
    "domain": "example.com",
    "notes": "Main bounce handler"
  }
]
```

Note: `password` is never returned.

---

### POST /api/bounces

Create or update a bounce account. **Requires authentication.**

Also creates the Linux system user and Maildir structure.

**Request:**
```json
{
  "id": 0,
  "username": "bounce1",
  "password": "secret123",
  "domain": "example.com",
  "notes": "Main bounce handler"
}
```

For updates, leave `password` empty to keep existing password.

---

### DELETE /api/bounces/{id}

Delete a bounce account from database. **Requires authentication.**

Note: Does not delete the system user.

---

### POST /api/bounces/apply

Ensure all bounce accounts exist as system users. **Requires authentication.**

**Response:**
```json
{
  "status": "ok"
}
```

---

## Configuration

### GET /api/config/preview

Preview generated KumoMTA configuration files. **Requires authentication.**

**Response:**
```json
{
  "sources_toml": "...",
  "queues_toml": "...",
  "listener_domains_toml": "...",
  "dkim_data_toml": "...",
  "init_lua": "..."
}
```

---

### POST /api/config/apply

Write configs, validate, and restart KumoMTA. **Requires authentication.**

**Response (success):**
```json
{
  "apply_result": {
    "sources_path": "/opt/kumomta/etc/policy/sources.toml",
    "queues_path": "/opt/kumomta/etc/policy/queues.toml",
    "listener_domains_path": "/opt/kumomta/etc/policy/listener_domains.toml",
    "dkim_data_path": "/opt/kumomta/etc/policy/dkim_data.toml",
    "init_lua_path": "/opt/kumomta/etc/policy/init.lua",
    "validation_ok": true,
    "validation_log": "",
    "restart_ok": true,
    "restart_log": ""
  }
}
```

**Response (error):**
```json
{
  "apply_result": { ... },
  "error": "kumod validation failed: ..."
}
```

---

## Logs

### GET /api/logs/kumomta?lines=100

Get KumoMTA service logs. **Requires authentication.**

### GET /api/logs/dovecot?lines=100

Get Dovecot service logs. **Requires authentication.**

### GET /api/logs/fail2ban?lines=100

Get Fail2Ban service logs. **Requires authentication.**

**Response:**
```json
{
  "service": "kumomta",
  "logs": "... log content ..."
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "description of the error"
}
```

Common HTTP status codes:
- `400` - Bad request (invalid input)
- `401` - Unauthorized (missing/invalid/expired token)
- `403` - Forbidden (action not allowed)
- `404` - Not found
- `500` - Internal server error
