# DNS Setup Guide

Complete DNS configuration required for KumoMTA UI to achieve maximum deliverability.

---

## Overview

KumoMTA UI uses **envelope-from separation** to isolate bounce handling per sender identity. This means each sender (e.g., `a1@domain.com`) gets its own subdomain (`a1.domain.com`) for the envelope return path. You must configure DNS records for **both** the main domain and each sender subdomain.

### How It Works

| Layer | Address | Purpose |
|-------|---------|---------|
| Header-From | `a1@domain.com` | What the recipient sees |
| Envelope-From | `a1@a1.domain.com` | Used for SPF checks and bounce routing |
| EHLO | `a1.domain.com` | Identifies the sending server to receivers |

---

## Prerequisites

Before configuring DNS, you need:
- Your **MTA server IP address** (e.g., `198.51.100.10`)
- Your **MTA hostname** (e.g., `mta.domain.com`)
- Access to your domain's DNS management panel
- Senders already created in the KumoMTA UI panel

---

## Step 1: MTA Server Records

These records identify your mail server itself.

```
# A record for MTA hostname
mta.domain.com.    IN  A       198.51.100.10

# Reverse DNS (PTR) - set via your hosting provider
198.51.100.10      IN  PTR     mta.domain.com.
```

> **Important:** The PTR (reverse DNS) record must match your MTA hostname exactly. Contact your hosting provider to set this — it cannot be configured in your domain's DNS panel.

---

## Step 2: Main Domain Records

For each domain you add in the panel (e.g., `domain.com`):

### A Record
```
domain.com.        IN  A       198.51.100.10
```

### MX Record
```
domain.com.        IN  MX  10  mta.domain.com.
```

### SPF Record
```
domain.com.        IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
```
> Use `-all` (hard fail) for production. This tells receivers that only your IP is authorized to send for this domain.

### DMARC Record
```
_dmarc.domain.com. IN  TXT     "v=DMARC1; p=reject; rua=mailto:dmarc@domain.com; ruf=mailto:dmarc@domain.com; adkim=r; aspf=r; pct=100"
```

| Parameter | Value | Meaning |
|-----------|-------|---------|
| `p=reject` | Policy | Reject emails that fail DMARC (strongest) |
| `adkim=r` | DKIM alignment | Relaxed — allows subdomain signing |
| `aspf=r` | SPF alignment | Relaxed — allows envelope subdomain |
| `pct=100` | Percentage | Apply to 100% of messages |
| `rua` | Aggregate reports | Email to receive daily DMARC reports |

> **Start with `p=none`** if this is a new domain. Move to `p=quarantine` after 1-2 weeks, then `p=reject` once you confirm no legitimate mail is failing.

### DKIM Record
The panel auto-generates DKIM keys. After creating a sender and running "Auto-Setup", get the DKIM public key from the DNS page in the panel.

```
default._domainkey.domain.com.  IN  TXT  "v=DKIM1; k=rsa; p=MIIBIjAN..."
```

Each sender gets its own DKIM selector. For sender `a1`:
```
a1._domainkey.domain.com.  IN  TXT  "v=DKIM1; k=rsa; p=MIIBIjAN..."
```

---

## Step 3: Sender Subdomain Records (Critical)

**This is the most commonly missed step.** Because KumoMTA UI uses envelope-from separation, each sender needs its own subdomain DNS records.

For sender `a1@domain.com`, the subdomain is `a1.domain.com`:

### A Record
```
a1.domain.com.     IN  A       198.51.100.10
```

### MX Record (for bounce handling)
```
a1.domain.com.     IN  MX  10  mta.domain.com.
```

### SPF Record
```
a1.domain.com.     IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
```

### Repeat for Each Sender

If you have senders `a1`, `news`, and `promo` on `domain.com`:

```
# a1.domain.com
a1.domain.com.     IN  A       198.51.100.10
a1.domain.com.     IN  MX  10  mta.domain.com.
a1.domain.com.     IN  TXT     "v=spf1 ip4:198.51.100.10 -all"

# news.domain.com
news.domain.com.   IN  A       198.51.100.10
news.domain.com.   IN  MX  10  mta.domain.com.
news.domain.com.   IN  TXT     "v=spf1 ip4:198.51.100.10 -all"

# promo.domain.com
promo.domain.com.  IN  A       198.51.100.10
promo.domain.com.  IN  MX  10  mta.domain.com.
promo.domain.com.  IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
```

> **Tip:** If your DNS provider supports wildcard records, you can simplify subdomain setup:
> ```
> *.domain.com.     IN  A       198.51.100.10
> *.domain.com.     IN  MX  10  mta.domain.com.
> *.domain.com.     IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
> ```
> Wildcard records apply to all subdomains that don't have explicit records.

---

## Step 4: Multiple IPs

If you use multiple sending IPs (e.g., different senders on different IPs), include all IPs in SPF:

```
domain.com.    IN  TXT  "v=spf1 ip4:198.51.100.10 ip4:198.51.100.11 ip4:198.51.100.12 -all"
```

Or use an IP range:
```
domain.com.    IN  TXT  "v=spf1 ip4:198.51.100.0/24 -all"
```

Each IP also needs its own PTR record pointing to your MTA hostname.

---

## Complete Example

For domain `example.com` with MTA hostname `mta.example.com` at IP `198.51.100.10`, and two senders (`info` and `news`):

```dns
; === MTA Server ===
mta.example.com.          IN  A       198.51.100.10

; === Main Domain ===
example.com.              IN  A       198.51.100.10
example.com.              IN  MX  10  mta.example.com.
example.com.              IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
_dmarc.example.com.       IN  TXT     "v=DMARC1; p=reject; rua=mailto:dmarc@example.com; adkim=r; aspf=r; pct=100"

; === DKIM (from panel) ===
info._domainkey.example.com.  IN  TXT  "v=DKIM1; k=rsa; p=<public-key-from-panel>"
news._domainkey.example.com.  IN  TXT  "v=DKIM1; k=rsa; p=<public-key-from-panel>"

; === Sender Subdomains ===
info.example.com.         IN  A       198.51.100.10
info.example.com.         IN  MX  10  mta.example.com.
info.example.com.         IN  TXT     "v=spf1 ip4:198.51.100.10 -all"

news.example.com.         IN  A       198.51.100.10
news.example.com.         IN  MX  10  mta.example.com.
news.example.com.         IN  TXT     "v=spf1 ip4:198.51.100.10 -all"
```

---

## Verification

After setting up DNS, verify your records using these tools:

### Command-Line Checks
```bash
# Check A record
dig +short domain.com A

# Check MX record
dig +short domain.com MX

# Check SPF
dig +short domain.com TXT

# Check DMARC
dig +short _dmarc.domain.com TXT

# Check DKIM (replace 'a1' with selector)
dig +short a1._domainkey.domain.com TXT

# Check sender subdomain SPF
dig +short a1.domain.com TXT

# Check reverse DNS
dig +short -x 198.51.100.10
```

### Online Tools
- [MXToolbox](https://mxtoolbox.com/) — Check MX, SPF, DKIM, DMARC, blacklists
- [mail-tester.com](https://www.mail-tester.com/) — Send a test email and get a deliverability score
- [DMARC Analyzer](https://dmarcanalyzer.com/) — Verify DMARC configuration

---

## Common Mistakes

| Mistake | Impact | Fix |
|---------|--------|-----|
| Missing sender subdomain SPF | SPF fails for envelope-from | Add SPF TXT for each `sender.domain.com` |
| Missing sender subdomain MX | Bounces can't be received | Add MX record for each subdomain |
| PTR doesn't match hostname | Rejected by Gmail/Microsoft | Contact hosting provider to set correct PTR |
| Using `~all` instead of `-all` | Weaker SPF enforcement | Use `-all` (hard fail) for production |
| DMARC `p=none` in production | No protection against spoofing | Upgrade to `p=reject` after testing |
| Missing DKIM for sender selector | DKIM signature fails | Run Auto-Setup in panel, add DNS TXT record |
| SPF record over 10 DNS lookups | SPF permerror | Use IP addresses directly, avoid `include:` chains |

---

## DNS Propagation

DNS changes can take up to 48 hours to propagate globally, though most changes are visible within 1-4 hours. After making changes:

1. Wait at least 1 hour
2. Verify using `dig` commands above
3. Send a test email to [mail-tester.com](https://www.mail-tester.com/) to confirm
4. Check the KumoMTA logs for any delivery errors
