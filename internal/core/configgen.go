package core

import (
	"fmt"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// Naming strategy (can be changed later in one place):

// Egress source name, unique per sender identity.
func SourceName(d models.Domain, s models.Sender) string {
	// Example: "example.com:info"
	return fmt.Sprintf("%s:%s", d.Name, s.LocalPart)
}

// Egress pool / tenant name per sender.
func PoolName(d models.Domain, s models.Sender) string {
	// Example: "example.com-info"
	return fmt.Sprintf("%s-%s", d.Name, s.LocalPart)
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
	b.WriteString("local listener_domains = kumo.toml_load('/opt/kumomta/etc/policy/listener_domains.toml')\n\n")

	// --- 3. Routing Logic ---
	b.WriteString(`
local function get_tenant_from_sender(sender_email)
  if sender_email then
    local localpart, domain = sender_email:match("([^@]+)@(.+)")
    if localpart and domain then
      return domain .. "-" .. localpart
    end
  end
  return "default"
end

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

kumo.on('get_egress_pool', function(pool_name)
  local source_key = pool_name:gsub("-", ":", 1)
  if sources_data[source_key] then
     return kumo.make_egress_pool {
       name = pool_name,
       entries = { { name = source_key } },
     }
  end
  return kumo.make_egress_pool { name = pool_name, entries = {} }
end)

kumo.on('get_egress_source', function(source_name)
  if sources_data[source_name] then
    local config = sources_data[source_name]
    return kumo.make_egress_source {
      name = source_name,
      source_address = config.source_address,
      ehlo_domain = config.ehlo_domain,
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

kumo.on('get_queue_config', function(domain, tenant, campaign, routing_domain)
  local tenant_key = 'tenant:' .. tenant
  local tenant_config = queues_data[tenant_key] or {}

  local params = {
    egress_pool = tenant_config.egress_pool or tenant,
    retry_interval = tenant_config.retry_interval or '1m',
    max_age = tenant_config.max_age or '3d',
    max_message_rate = tenant_config.max_message_rate,
  }
  return kumo.make_queue_config(params)
end)

local function sign_with_dkim(msg)
  local sender = msg:from_header()
  if not sender then return end
  
  local sender_email = sender.email:lower()
  local sender_domain = sender.domain:lower()

  local domain_key = 'domain.' .. sender_domain
  local domain_config = dkim_data[domain_key]

  if not domain_config then return end

  if domain_config.policy then
    for _, policy in ipairs(domain_config.policy) do
      if policy.match_sender and sender_email == policy.match_sender:lower() then
        local signer = kumo.dkim.rsa_sha256_signer {
          domain = sender_domain,
          selector = policy.selector,
          headers = domain_config.headers or { 'From', 'To', 'Subject', 'Date', 'Message-ID', 'List-Unsubscribe' },
          key = { key_file = policy.filename },
        }
        msg:dkim_sign(signer)
        return
      end
    end
  end
end

-- --- HEADER SCRUBBER & SIGNER ---
local function scrub_headers(msg)
  -- 1. Remove identifying headers using correct API
  msg:remove_all_named_headers('User-Agent')
  msg:remove_all_named_headers('X-Mailer')
  msg:remove_all_named_headers('X-Originating-IP')
  msg:remove_all_named_headers('X-Report-Abuse') 
  msg:remove_all_named_headers('X-EBS')
  
  -- We do NOT remove X-RefID (our renamed KumoRef) to keep bounces working
  msg:remove_x_headers { 'x-campaign', 'x-tenant', 'x-kumomta' }
end

local function inject_fake_received(msg)
  -- Add GENERIC Received Header (Fake Postfix style)
  local remote_ip = msg:get_meta('received_from_ip') or '127.0.0.1'
  local timestamp = os.date("%a, %d %b %Y %H:%M:%S %z")
  
  msg:prepend_header('Received', string.format(
    "from %s ([%s])\r\n\tby %s (Postfix) with ESMTPS\r\n\tfor <%s>; %s",
    msg:get_meta('received_from_name') or 'localhost',
    remote_ip,
    '` + mainHostname + `',
    msg:recipient(),
    timestamp
  ))
end

kumo.on('smtp_server_message_received', function(msg)
  -- 1. Extract Metadata FIRST
  local sender = msg:from_header()
  local sender_email = sender and sender.email or ""

  local tenant = get_tenant_from_sender(sender_email)
  msg:set_meta('tenant', tenant)

  local campaign = msg:get_first_named_header_value('X-Campaign')
  if campaign then msg:set_meta('campaign', campaign) end

  -- 2. Scrub Headers & Add Fake Received
  scrub_headers(msg)
  inject_fake_received(msg)

  -- 3. Sign with DKIM
  sign_with_dkim(msg)
end)

kumo.on('http_message_generated', function(msg)
  -- 1. Extract Metadata
  local tenant = msg:get_first_named_header_value('X-Tenant')
  if not tenant then
    local sender = msg:from_header()
    local sender_email = sender and sender.email or ""
    tenant = get_tenant_from_sender(sender_email)
  end
  msg:set_meta('tenant', tenant)
  
  local campaign = msg:get_first_named_header_value('X-Campaign')
  if campaign then msg:set_meta('campaign', campaign) end

  -- 2. Scrub Headers & Add Fake Received
  scrub_headers(msg)
  inject_fake_received(msg)

  -- 3. Sign
  sign_with_dkim(msg)
end)

-- Optional: Custom Hook (Safe place for manual overrides)
pcall(dofile, '/opt/kumomta/etc/policy/custom.lua')
`)

	return b.String()
}
