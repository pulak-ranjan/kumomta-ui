# Deliverability Guide

How KumoMTA UI maximizes email deliverability through envelope separation, ISP traffic shaping, bounce management, and reputation protection.

---

## Table of Contents

1. [Envelope-From Separation](#envelope-from-separation)
2. [ISP Traffic Shaping](#isp-traffic-shaping)
3. [Bounce Management](#bounce-management)
4. [DKIM Signing](#dkim-signing)
5. [Header Scrubbing](#header-scrubbing)
6. [IP Warmup](#ip-warmup)
7. [Connection to External Senders (MailWizz, etc.)](#connection-to-external-senders)
8. [Settings Reference](#settings-reference)
9. [Troubleshooting](#troubleshooting)

---

## Envelope-From Separation

### What It Is

Every email has two "from" addresses:
- **Header-From** (`From:` header): What the recipient sees in their email client
- **Envelope-From** (`MAIL FROM`): Used by receiving servers for SPF checks and bounce routing

KumoMTA UI automatically separates these to isolate bounce handling per sender identity.

### How It Works

```
Header-From:   newsletter@domain.com    (visible to recipient)
Envelope-From: newsletter@newsletter.domain.com  (used for SPF + bounces)
EHLO Domain:   newsletter.domain.com    (server identification)
```

### Why This Matters

| Benefit | Explanation |
|---------|-------------|
| **Bounce isolation** | Bounces from `newsletter` don't affect `transactional` sender reputation |
| **Per-sender SPF** | Each sender subdomain has its own SPF record, so SPF alignment is clean |
| **Reputation separation** | ISPs track reputation per subdomain — one sender's bad list doesn't poison others |
| **DMARC compliance** | With `aspf=r` (relaxed), `newsletter.domain.com` aligns with `domain.com` |

### DNS Requirements

Each sender subdomain needs its own DNS records. See [DNS-SETUP.md](DNS-SETUP.md#step-3-sender-subdomain-records-critical) for details.

---

## ISP Traffic Shaping

### What It Is

Major ISPs (Gmail, Microsoft, Yahoo) enforce strict rate limits on incoming mail. Exceeding these limits results in temporary blocks (421/450 deferrals) or permanent blocks (550 rejections). KumoMTA UI applies conservative per-ISP limits automatically.

### Default Limits

| ISP | Messages/Hour | Connections | Msgs/Connection | Conn Rate |
|-----|---------------|-------------|-----------------|-----------|
| **Gmail** (google.com) | 50/h | 3 | 20 | 5/min |
| **Microsoft** (outlook.com, hotmail.com, live.com) | 50/h | 2 | 10 | 3/min |
| **Yahoo/AOL** (yahoodns.net) | 100/h | 3 | 20 | 5/min |
| **All others** (default) | No ISP limit | 5 | 50 | 10/min |

### How Matching Works

Limits are matched against the **MX hostname** (site_name), not the recipient domain:
- `user@gmail.com` resolves to MX `gmail-smtp-in.l.google.com` → matches `google.com` pattern
- `user@outlook.com` resolves to MX `*.protection.outlook.com` → matches `outlook.com` pattern
- `user@yahoo.com` resolves to MX `*.yahoodns.net` → matches `yahoodns.net` pattern

### Warmup Considerations

For **new IPs**, these limits should be even lower. Start with:
- **Week 1-2:** 10-20 emails/hour to major ISPs
- **Week 3-4:** 50/hour
- **Month 2+:** Gradually increase based on delivery rates

Use the warmup plans in the panel (Settings > Sender > Warmup Plan) to control sending rate.

---

## Bounce Management

### Bounce Server Setup

KumoMTA UI manages bounce handling through bounce accounts — Linux system users that receive bounced emails.

#### How to Set Up

1. **Create a bounce account** in the panel:
   - Go to **Bounce Accounts** page
   - Click **Create Bounce Account**
   - Set username (e.g., `b-newsletter`), domain, and password
   - The panel creates a Linux system user automatically

2. **Apply system state:**
   - Click **Apply** to ensure all bounce accounts exist as Linux users

3. **Bounce routing:**
   - The envelope-from (`newsletter@newsletter.domain.com`) directs bounces back to your server
   - The MX record on `newsletter.domain.com` points to your MTA hostname
   - KumoMTA processes the bounce and logs it

#### Bounce Classification

KumoMTA uses IANA bounce classification rules (`/opt/kumomta/share/bounce_classifier/iana.toml`) to categorize bounces:

| Type | Example | Action |
|------|---------|--------|
| Hard bounce (5xx) | `550 User not found` | Remove address permanently |
| Soft bounce (4xx) | `450 Try again later` | Retry up to `max_age` (3 days) |
| Block bounce | `550 Blocked by policy` | Check reputation, reduce volume |

### Retry Configuration

| Setting | Value | Meaning |
|---------|-------|---------|
| `retry_interval` | 5 minutes | Wait between retry attempts |
| `max_age` | 3 days | Stop retrying after this period |

---

## DKIM Signing

### How It Works

Each sender gets identity-based DKIM signing:

1. **Auto-Setup** generates RSA-2048 DKIM keys per sender
2. Keys are stored at `/opt/kumomta/etc/dkim/{domain}/{localpart}.key`
3. Signing uses the sender's localpart as the DKIM selector
4. The config matches `From:` header email to the correct key

### Signed Headers

The following headers are included in the DKIM signature:

```
From, To, Subject, Date, Message-ID, List-Unsubscribe, List-Unsubscribe-Post
```

> `List-Unsubscribe-Post` is included because Gmail requires it for one-click unsubscribe functionality. If the sending application includes this header, it will be signed and protected.

### Verification

After DNS setup, verify DKIM is working:

```bash
# Send a test email
swaks --to test@gmail.com --from newsletter@domain.com \
  --server mta.domain.com:587 \
  --auth LOGIN --auth-user newsletter@domain.com --auth-password "pass" \
  --tls

# Check the email headers for:
# dkim=pass header.d=domain.com header.s=newsletter
```

---

## Header Scrubbing

KumoMTA UI automatically removes headers that expose internal infrastructure:

| Removed Header | Reason |
|----------------|--------|
| `User-Agent` | Reveals sending software |
| `X-Mailer` | Reveals sending software |
| `X-Originating-IP` | Exposes internal IPs |
| `X-Report-Abuse` | Internal header |
| `X-EBS` | Internal header |
| `X-Campaign`, `X-Tenant`, `X-KumoMTA` | Internal routing headers |

A clean `Received` header is injected that mimics standard Postfix format, preventing fingerprinting of your MTA software.

---

## IP Warmup

### Why Warmup Matters

New IPs have no sending reputation. ISPs will throttle or block mail from unknown IPs. Gradual volume increase builds positive reputation.

### Warmup Plans

The panel offers warmup plans in **Settings > Sender > Warmup Plan**:

| Plan | Day 1 | Day 7 | Day 14 | Day 30 | Best For |
|------|-------|-------|--------|--------|----------|
| Conservative | 50/hr | 200/hr | 500/hr | 2000/hr | Most senders |
| Moderate | 100/hr | 500/hr | 1000/hr | 5000/hr | Established brands |
| Aggressive | 500/hr | 2000/hr | 5000/hr | 10000/hr | Transactional only |

### Warmup Best Practices

1. **Start with engaged users** — Send to your most active subscribers first
2. **Monitor bounce rates** — Keep hard bounces below 2%
3. **Watch for deferrals** — 421/450 responses mean you're sending too fast
4. **Check blacklists** — Use the panel's RBL monitoring feature
5. **Maintain consistency** — Send similar volumes daily, avoid spikes

---

## Connection to External Senders

### MailWizz / Mautic / Other SMTP Clients

To connect an external email application to KumoMTA:

#### SMTP Settings

| Setting | Value |
|---------|-------|
| **SMTP Host** | Your MTA hostname (e.g., `mta.domain.com`) |
| **SMTP Port** | `587` (STARTTLS) or `465` (Implicit TLS) |
| **Encryption** | TLS/SSL or STARTTLS |
| **Authentication** | LOGIN or PLAIN |
| **Username** | Sender email (e.g., `newsletter@domain.com`) |
| **Password** | The SMTP password set in the panel |

#### Port Reference

| Port | Protocol | Encryption | Use Case |
|------|----------|------------|----------|
| 25 | SMTP | STARTTLS (optional) | Server-to-server relay from trusted IPs |
| 587 | Submission | STARTTLS (required) | Authenticated sending from external apps |
| 465 | SMTPS | Implicit TLS | Authenticated sending (legacy, some clients prefer) |

#### TLS Certificate Setup

TLS certificates are required for ports 587 and 465. The installer sets up Let's Encrypt for the web panel. To configure TLS for SMTP:

1. Go to **Settings** in the panel
2. Set **TLS Certificate Path** to your certificate file (e.g., `/etc/letsencrypt/live/mta.domain.com/fullchain.pem`)
3. Set **TLS Private Key Path** to your key file (e.g., `/etc/letsencrypt/live/mta.domain.com/privkey.pem`)
4. Click **Apply Config** to regenerate and restart KumoMTA

> **Note:** If TLS certificates are not configured, port 465 (implicit TLS) will not start, and AUTH will not be offered on ports 25/587 (most clients require TLS before sending credentials).

#### Testing Connection

```bash
# Test STARTTLS on port 587
swaks --to test@example.com --from sender@domain.com \
  --server mta.domain.com:587 \
  --auth LOGIN --auth-user sender@domain.com --auth-password "yourpassword" \
  --tls

# Test implicit TLS on port 465
swaks --to test@example.com --from sender@domain.com \
  --server mta.domain.com:465 \
  --auth LOGIN --auth-user sender@domain.com --auth-password "yourpassword" \
  --tlsc

# Expected output: "250 OK" at the end
```

---

## Settings Reference

### Panel Settings (Settings Page)

| Setting | Description | Example |
|---------|-------------|---------|
| **Main Hostname** | Your MTA server's FQDN | `mta.domain.com` |
| **Main Server IP** | Primary sending IP | `198.51.100.10` |
| **Relay IPs** | IPs allowed to relay without auth (comma-separated) | `10.0.0.5, 10.0.0.6` |
| **TLS Certificate Path** | Path to SSL certificate | `/etc/letsencrypt/live/mta.domain.com/fullchain.pem` |
| **TLS Private Key Path** | Path to SSL private key | `/etc/letsencrypt/live/mta.domain.com/privkey.pem` |

### Generated Config Files

When you click **Apply Config**, these files are generated at `/opt/kumomta/etc/policy/`:

| File | Purpose |
|------|---------|
| `init.lua` | Main KumoMTA policy — listeners, auth, hooks, routing |
| `auth.toml` | SMTP credentials (`user@domain.com = "password"`) |
| `sources.toml` | Egress sources with IP and EHLO domain per sender |
| `queues.toml` | Queue config per tenant (retry, rate limits) |
| `listener_domains.toml` | Accepted domains and subdomains for relay |
| `dkim_data.toml` | DKIM keys, selectors, and signing policies |

### Queue Settings (per sender)

| Setting | Default | Description |
|---------|---------|-------------|
| `retry_interval` | 5 minutes | Time between delivery retries |
| `max_age` | 3 days | Maximum time to keep retrying |
| `max_message_rate` | Per warmup plan | Messages per time unit |

---

## Troubleshooting

### Common Issues

#### "535 Authentication failed"
- Verify the SMTP username is the full email (e.g., `sender@domain.com`)
- Verify the password matches what's set in the panel
- Ensure TLS is configured — AUTH is not offered without TLS
- Check KumoMTA logs: `journalctl -u kumomta -f`

#### "550 Relaying not permitted"
- Ensure the sender is authenticated (use port 587 with TLS + credentials)
- Check that the domain or subdomain exists in listener_domains.toml
- Authenticated users are automatically allowed to relay — verify auth succeeded first

#### "SPF softfail/fail" in email headers
- Verify the sender subdomain has an SPF record (e.g., `a1.domain.com TXT "v=spf1 ip4:... -all"`)
- The envelope-from subdomain is what's checked for SPF, not the header-from domain
- Run: `dig +short a1.domain.com TXT` to verify

#### "DKIM signature invalid"
- Ensure the DKIM DNS record matches the public key generated by the panel
- Check the selector — each sender uses their localpart as selector (e.g., `a1._domainkey.domain.com`)
- Verify the DKIM key file exists: `ls /opt/kumomta/etc/dkim/domain.com/a1.key`

#### Low mail-tester.com score
1. Check all DNS records are set (SPF, DKIM, DMARC for main domain AND subdomains)
2. Verify PTR record matches MTA hostname
3. Ensure `List-Unsubscribe` header is present (provided by your sending application)
4. Check IP reputation at [SenderScore](https://senderscore.org/) and [Google Postmaster](https://postmaster.google.com/)
5. Verify no blacklist issues via the panel's RBL monitoring

#### Messages stuck in queue
- Check queue status in the panel (Queue page)
- Look at KumoMTA logs for deferral reasons: `journalctl -u kumomta | grep "4[0-9][0-9]"`
- If ISP is throttling, the traffic shaping limits will handle retries automatically
- Use **Flush Queue** to force immediate retry of deferred messages
