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

// GenerateSourcesTOML builds the sources.toml content based on Snapshot.
func GenerateSourcesTOML(snap *Snapshot) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		// Section comment for readability
		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s Sources\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		for _, s := range d.Senders {
			name := SourceName(d, s)

			// You might want EHLO host to be "<localpart>.<domain>" or "mail.<domain>"
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

// GenerateQueuesTOML builds queues.toml content.
func GenerateQueuesTOML(snap *Snapshot) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s Tenants (per sender)\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		for _, s := range d.Senders {
			pool := PoolName(d, s)
			tenantKey := fmt.Sprintf("tenant:%s", pool)

			fmt.Fprintf(&b, "[\"%s\"]\n", tenantKey)
			fmt.Fprintf(&b, "egress_pool = \"%s\"\n", pool)
			fmt.Fprintf(&b, "retry_interval = \"1m\"\n")
			fmt.Fprintf(&b, "max_age = \"3d\"\n\n")
		}
	}

	return b.String()
}

// =============================
// listener_domains.toml generator
// =============================

// GenerateListenerDomainsTOML builds listener_domains.toml content.
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
//
// NOTE: This assumes you generate DKIM keys separately and store them
// under a consistent path, e.g. /opt/kumomta/etc/dkim/<domain>/<localpart>.key
//
// This generator only maps sender -> key file.
//

func GenerateDKIMDataTOML(snap *Snapshot, dkimBasePath string) string {
	var b strings.Builder

	for _, d := range snap.Domains {
		if len(d.Senders) == 0 {
			continue
		}

		fmt.Fprintf(&b, "# ========================================\n")
		fmt.Fprintf(&b, "# %s DKIM Configuration\n", d.Name)
		fmt.Fprintf(&b, "# ========================================\n\n")

		fmt.Fprintf(&b, "[domain.\"%s\"]\n", d.Name)
		fmt.Fprintf(&b, "selector = \"default\"\n")
		fmt.Fprintf(&b, "headers = [\"From\", \"To\", \"Subject\", \"Date\", \"Message-ID\"]\n\n")

		for _, s := range d.Senders {
			selector := s.LocalPart // or use "default" for all, up to you
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
// init.lua generator (basic skeleton)
// =======================

// GenerateInitLua creates a basic init.lua with HTTP + SMTP listeners.
// This is still generic; details like ports and relay_ips are fetched
// from Settings inside Snapshot.
func GenerateInitLua(snap *Snapshot) string {
	// Safe defaults if settings are missing
	mainHostname := "localhost"
	relayIPs := []string{"127.0.0.1"}

	if snap.Settings != nil {
		if snap.Settings.MainHostname != "" {
			mainHostname = snap.Settings.MainHostname
		}
		if snap.Settings.MailWizzIP != "" {
			// MailWizzIP is actually generic "relay IPs" CSV
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

  kumo.start_esmtp_listener {
    listen = '0.0.0.0:25',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    relay_hosts = { '127.0.0.1' },
  }

  kumo.start_esmtp_listener {
    listen = '0.0.0.0:587',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    relay_hosts = { `)
	b.WriteString(relayListStr)
	b.WriteString(` },
  }

  kumo.start_esmtp_listener {
    listen = '0.0.0.0:465',
    hostname = '`)
	b.WriteString(mainHostname)
	b.WriteString(`',
    relay_hosts = { `)
	b.WriteString(relayListStr)
	b.WriteString(` },
  }
end)

`)

	// NOTE: here we will later include dynamic logic for:
	// - get_listener_domain
	// - get_egress_pool
	// - get_egress_source
	// - get_queue_config
	// - DKIM signing

	b.WriteString("-- TODO: load TOML files (sources, queues, dkim_data, listener_domains)\n")
	b.WriteString("-- and define the rest of policy callbacks here.\n")

	return b.String()
}
