package core

import (
	"fmt"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// Naming strategy (can be changed later in one place):

// Egress source name, unique per sender identity.
// FIX: Using double-underscore (__) as separator because KumoMTA interprets
// colons in queue/tenant names as internal delimiters.
func SourceName(d models.Domain, s models.Sender) string {
	// Example: "example.com__info"
	return fmt.Sprintf("%s__%s", d.Name, s.LocalPart)
}

// Egress pool / tenant name per sender.
// FIX: Using double-underscore (__) as separator
// - Hyphen (-) breaks hyphenated domains like my-domain.com
// - Colon (:) is interpreted by KumoMTA as internal delimiter
// - Double-underscore (__) is safe and unambiguous
func PoolName(d models.Domain, s models.Sender) string {
	// Example: "example.com__info" (same as SourceName for consistency)
	return fmt.Sprintf("%s__%s", d.Name, s.LocalPart)
}

// =======================
// auth.toml generator (SMTP Authentication)
// =======================

func GenerateAuthTOML(snap *Snapshot) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# KumoMTA SMTP Authentication Credentials")
	fmt.Fprintln(&b, "# Format: username = \"password\"")
	fmt.Fprintln(&b, "")

	for _, d := range snap.Domains {
		for _, s := range d.Senders {
			// Only add if a password is set
			if s.SMTPPassword != "" {
				// Escape quotes in password just in case
				safePass := strings.ReplaceAll(s.SMTPPassword, "\"", "\\\"")
				fmt.Fprintf(&b, "\"%s\" = \"%s\"\n", s.Email, safePass)
			}
		}
	}
	return b.String()
}

// =======================
// sources.toml generator
// =======================

func GenerateSourcesTOML(snap *Snapshot) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s Sources\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		for _, s := range d.Senders {
			name := SourceName(d, s)
			// EHLO host: "localpart.domain" or "mail.domain"
			ehloDomain := fmt.Sprintf("%s.%s", s.LocalPart, d.Name)

			fmt.Fprintf(&b, "[\"%s\"]\n", name)
			fmt.Fprintf(&b, "source_address = \"%s\"\n", s.IP)
			fmt.Fprintf(&b, "ehlo_domain = \"%s\"\n\n", ehloDomain)
		}
	}
	return b.String()
}

// =======================
// queues.toml generator
// =======================

func GenerateQueuesTOML(snap *Snapshot) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s Tenants\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		for _, s := range d.Senders {
			pool := PoolName(d, s)
			tenantKey := fmt.Sprintf("tenant:%s", pool)

			fmt.Fprintf(&b, "[\"%s\"]\n", tenantKey)
			fmt.Fprintf(&b, "egress_pool = \"%s\"\n", pool)
			fmt.Fprintf(&b, "retry_interval = \"1m\"\n")
			fmt.Fprintf(&b, "max_age = \"3d\"\n")

			rate := GetSenderRate(s)
			if rate != "" {
				fmt.Fprintf(&b, "max_message_rate = \"%s\"\n", rate)
			}
			fmt.Fprintf(&b, "\n")
		}
	}
	return b.String()
}

// =============================
// listener_domains.toml generator
// =============================

func GenerateListenerDomainsTOML(snap *Snapshot) string {
	var b strings.Builder
	for _, d := range snap.Domains {
		fmt.Fprintf(&b, "[\"%s\"]\n", d.Name)
		fmt.Fprintf(&b, "relay_to = true\n")
		fmt.Fprintf(&b, "log_oob = true\n")
		fmt.Fprintf(&b, "log_arf = true\n\n")
	}
	return b.String()
}

// =======================
// dkim_data.toml generator
// =======================

func GenerateDKIMDataTOML(snap *Snapshot, dkimBasePath string) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s DKIM\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		fmt.Fprintf(&b, "[domain.\"%s\"]\n", d.Name)
		fmt.Fprintf(&b, "selector = \"default\"\n")
		// DO NOT include X- headers here, or scrubbing them will break the signature
		fmt.Fprintf(&b, "headers = [\"From\", \"To\", \"Subject\", \"Date\", \"Message-ID\", \"List-Unsubscribe\"]\n\n")

		for _, s := range d.Senders {
			selector := s.LocalPart
			keyFile := fmt.Sprintf("%s/%s/%s.key", strings.TrimRight(dkimBasePath, "/"), d.Name, s.LocalPart)
			matchSender := s.Email

			fmt.Fprintf(&b, "[[domain.\"%s\".policy]]\n", d.Name)
			fmt.Fprintf(&b, "selector = \"%s\"\n", selector)
			fmt.Fprintf(&b, "filename = \"%s\"\n", keyFile)
			fmt.Fprintf(&b, "match_sender = \"%s\"\n\n", matchSender)
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String()
}

// =======================
// init.lua generator
// =======================

func GenerateInitLua(snap *Snapshot) string {
	mainHostname := "localhost"
	relayIPs := []string{"127.0.0.1"}
	listenAddr := "127.0.0.1:25"

	if snap.Settings != nil {
		if snap.Settings.MainHostname != "" {
			mainHostname = snap.Settings.MainHostname
		}
		if snap.Settings.SMTPListenAddr != "" {
			listenAddr = snap.Settings.SMTPListenAddr
		}
		if snap.Settings.MailWizzIP != "" {
			parts := strings.Split(snap.Settings.MailWizzIP, ",")
			relayIPs = []string{"127.0.0.1"}
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					relayIPs = append(relayIPs, p)
				}
			}
		}
	}

	relayList := make([]string, 0, len(relayIPs))
	for _, ip := range relayIPs {
		relayList = append(relayList, fmt.Sprintf("'%s'", ip))
	}
	relayListStr := strings.Join(relayList, ", ")

	var b strings.Builder

	// --- 1. System Config ---
	b.WriteString(`local kumo = require 'kumo'

kumo.on('init', function()
  kumo.define_spool {
    name = 'data',
    path = '/var/spool/kumomta/data',
    kind = 'LocalDisk',
  }

  kumo.define_spool {
    name = 'meta',
    path = '/var/spool/kumomta/meta',
    kind = 'LocalDisk',
  }

  kumo.configure_local_logs {
    log_dir = '/var/log/kumomta',
    max_segment_duration = '10 seconds',
  }

  kumo.configure_bounce_classifier {
    files = {
      '/opt/kumomta/share/bounce_classifier/iana.toml',
    },
  }

  kumo.start_http_listener {
    listen = '127.0.0.1:8000',
    use_tls = false,
    trusted_hosts = { '127.0.0.1' },
  }

  -- Define Stealth Trace Settings
  local trace_settings = {
    received_header = false,      -- Disable default KumoMTA Received header
    supplemental_header = true,   -- Keep tracking enabled
    header_name = 'X-RefID',      -- Rename header to hide "Kumo"
  }

  -- SMTP Listeners
  kumo.start_esmtp_listener {
    listen = '`)
	b.WriteString(listenAddr)
	b.WriteString(`',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    banner = '220 ' .. '`)
	b.WriteString(mainHostname)
	b.WriteString(`', -- Minimal banner
    relay_hosts = { `)
	b.WriteString(relayListStr)
	b.WriteString(` },
    trace_headers = trace_settings,
  }

  kumo.start_esmtp_listener {
    listen = '0.0.0.0:587',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    banner = '220 ' .. '`)
	b.WriteString(mainHostname)
	b.WriteString(`', 
    relay_hosts = { `)
	b.WriteString(relayListStr)
	b.WriteString(` },
    trace_headers = trace_settings,
  }

  kumo.start_esmtp_listener {
    listen = '0.0.0.0:465',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    banner = '220 ' .. '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    relay_hosts = { `)
	b.WriteString(relayListStr)
	b.WriteString(` },
    trace_headers = trace_settings,
  }
end)

`)

	// --- 2. Load Policy Data ---
	b.WriteString("-- Load config files (generated from DB)\n")
	b.WriteString("local sources_data = kumo.toml_load('/opt/kumomta/etc/policy/sources.toml')\n")
	b.WriteString("local queues_data = kumo.toml_load('/opt/kumomta/etc/policy/queues.toml')\n")
	b.WriteString("local dkim_data = kumo.toml_load('/opt/kumomta/etc/policy/dkim_data.toml')\n")
	b.WriteString("local listener_domains = kumo.toml_load('/opt/kumomta/etc/policy/listener_domains.toml')\n")
	b.WriteString("local auth_users = kumo.toml_load('/opt/kumomta/etc/policy/auth.toml')\n\n")

	// --- 3. SMTP Authentication Hook ---
	b.WriteString(`-- =====================================================
-- SMTP AUTHENTICATION (PLAIN)
-- =====================================================
kumo.on('smtp_server_auth_plain', function(auth_user, auth_password)
  local valid_pass = auth_users[auth_user]
  if valid_pass and valid_pass == auth_password then
    return true
  end
  return false
end)

`)

	// --- 4. Tenant Logic (Double-Underscore Separator) ---
	b.WriteString(`-- =====================================================
-- TENANT LOGIC
-- =====================================================
-- FIX: Using double-underscore (__) separator because:
-- - Hyphen (-) breaks hyphenated domains like my-domain.com
-- - Colon (:) is interpreted by KumoMTA as internal delimiter
-- - Double-underscore (__) is safe and unambiguous
-- Example: editor@my-domain.com -> tenant = "my-domain.com__editor"
local function get_tenant_from_sender(sender_email)
  if sender_email then
    local localpart, domain = sender_email:match("([^@]+)@(.+)")
    if localpart and domain then
      return domain .. "__" .. localpart
    end
  end
  return "default"
end

-- =====================================================
-- LISTENER DOMAIN CONFIG
-- =====================================================
kumo.on('get_listener_domain', function(domain, listener, conn_meta)
  if listener_domains[domain] then
    local config = listener_domains[domain]
    return kumo.make_listener_domain {
      relay_to = config.relay_to or false,
      log_oob = config.log_oob or false,
      log_arf = config.log_arf or false,
    }
  end
  return kumo.make_listener_domain { relay_to = false }
end)

-- =====================================================
-- EGRESS POOLS / SOURCES
-- =====================================================
-- Pool name uses double-underscore separator: "domain.com__localpart"
kumo.on('get_egress_pool', function(pool_name)
  -- Pool name format: "domain.com__localpart" (same as source name)
  if sources_data[pool_name] then
    return kumo.make_egress_pool {
      name = pool_name,
      entries = { { name = pool_name } },
    }
  end
  return kumo.make_egress_pool { name = pool_name, entries = {} }
end)

kumo.on('get_egress_source', function(source_name)
  local cfg = sources_data[source_name]
  if cfg then
    return kumo.make_egress_source {
      name = source_name,
      source_address = cfg.source_address,
      ehlo_domain = cfg.ehlo_domain,
    }
  end
  return kumo.make_egress_source { name = source_name }
end)

kumo.on('get_egress_path_config', function(domain, egress_source, site_name)
  return kumo.make_egress_path {
    enable_tls = 'OpportunisticInsecure',
    enable_mta_sts = false,
  }
end)

-- =====================================================
-- QUEUE CONFIG
-- =====================================================
kumo.on('get_queue_config', function(domain, tenant, campaign, routing_domain)
  tenant = tenant or "default"
  local cfg = queues_data['tenant:' .. tenant] or {}
  return kumo.make_queue_config {
    egress_pool = cfg.egress_pool or tenant,
    retry_interval = cfg.retry_interval or '1m',
    max_age = cfg.max_age or '3d',
    max_message_rate = cfg.max_message_rate,
  }
end)

-- =====================================================
-- DKIM SIGNING (IDENTITY-BASED)
-- =====================================================
local function dkim_sign_message(msg)
  local sender = msg:from_header()
  if not sender then
    kumo.log_error("DKIM: missing From header")
    return
  end

  local sender_email = sender.email:lower()
  local sender_domain = sender.domain:lower()

  local domain_cfg = dkim_data.domain[sender_domain]
  if not domain_cfg or not domain_cfg.policy then
    kumo.log_error("DKIM: no DKIM config for domain " .. sender_domain)
    return
  end

  for _, policy in ipairs(domain_cfg.policy) do
    if sender_email == policy.match_sender:lower() then
      msg:dkim_sign(kumo.dkim.rsa_sha256_signer {
        domain = sender_domain,
        selector = policy.selector,
        headers = domain_cfg.headers,
        key = policy.filename,
      })
      return
    end
  end

  kumo.log_error("DKIM: no identity match for " .. sender_email)
end

-- =====================================================
-- HEADER SCRUBBING + SAFE RECEIVED HEADER
-- =====================================================
local function scrub_headers(msg)
  msg:remove_all_named_headers('User-Agent')
  msg:remove_all_named_headers('X-Mailer')
  msg:remove_all_named_headers('X-Originating-IP')
  msg:remove_all_named_headers('X-Report-Abuse')
  msg:remove_all_named_headers('X-EBS')
  msg:remove_x_headers { 'x-campaign', 'x-tenant', 'x-kumomta' }

  local remote_ip = msg:get_meta('received_from_ip') or '127.0.0.1'
  local timestamp = os.date("%a, %d %b %Y %H:%M:%S %z")
  local rcpt = msg:recipient() or "unknown"

  msg:prepend_header('Received', string.format(
    "from %s ([%s])\r\n\tby %s (Postfix) with ESMTPS\r\n\tfor <%s>; %s",
    msg:get_meta('received_from_name') or 'localhost',
    remote_ip,
    '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    rcpt,
    timestamp
  ))
end

-- =====================================================
-- SMTP PATH
-- =====================================================
kumo.on('smtp_server_message_received', function(msg)
  local sender = msg:from_header()
  local sender_email = sender and sender.email or ""

  local tenant = get_tenant_from_sender(sender_email)
  msg:set_meta('tenant', tenant)

  local campaign = msg:get_first_named_header_value('X-Campaign')
  if campaign then msg:set_meta('campaign', campaign) end

  scrub_headers(msg)
  dkim_sign_message(msg)
end)

-- =====================================================
-- HTTP / API PATH
-- =====================================================
kumo.on('http_message_generated', function(msg)
  local tenant = msg:get_first_named_header_value('X-Tenant')
  if not tenant then
    local sender = msg:from_header()
    local sender_email = sender and sender.email or ""
    tenant = get_tenant_from_sender(sender_email)
  end
  msg:set_meta('tenant', tenant)

  local campaign = msg:get_first_named_header_value('X-Campaign')
  if campaign then msg:set_meta('campaign', campaign) end

  scrub_headers(msg)
  dkim_sign_message(msg)
end)

-- =====================================================
-- OPTIONAL: Custom Hook (Safe place for manual overrides)
-- =====================================================
pcall(dofile, '/opt/kumomta/etc/policy/custom.lua')
`)

	return b.String()
}
