package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
	"gorm.io/gorm/clause"
)

// Standard KumoMTA paths
const (
	SourcesPath  = "/opt/kumomta/etc/policy/sources.toml"
	InitLuaPath  = "/opt/kumomta/etc/policy/init.lua"
	DBPath       = "/var/lib/kumomta-ui/panel.db"
)

func main() {
	// 1. Safety Check: Verify KumoMTA config exists
	if _, err := os.Stat(SourcesPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  KumoMTA configuration not found at %s\n", SourcesPath)
		fmt.Println("   Skipping migration (Run this only on a server with KumoMTA installed)")
		return
	}

	// 2. Open Database
	fmt.Printf("📂 Opening DB at %s...\n", DBPath)
	st, err := store.NewStore(DBPath)
	if err != nil {
		log.Fatalf("❌ Failed to open DB: %v", err)
	}

	// 3. Parse Global Settings (init.lua)
	parseInitLua(st)

	// 4. Parse Domains, Senders & IPs (sources.toml)
	parseSourcesToml(st)

	fmt.Println("\n✅ Migration complete! Configuration and IPs have been imported.")
}

func parseInitLua(st *store.Store) {
	fmt.Println("🔍 Reading init.lua for settings...")
	file, err := os.Open(InitLuaPath)
	if err != nil {
		fmt.Printf("⚠️  Could not read init.lua: %v\n", err)
		return
	}
	defer file.Close()

	settings := &models.AppSettings{
		AIProvider: "openai", // Default
	}

	// Regex to find config values generic format
	reHostname := regexp.MustCompile(`hostname\s*=\s*'([^']+)'`)
	reRelay := regexp.MustCompile(`relay_hosts\s*=\s*\{\s*'([^']+)'`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Capture Hostname
		if matches := reHostname.FindStringSubmatch(line); len(matches) > 1 {
			if settings.MainHostname == "" {
				settings.MainHostname = matches[1]
				fmt.Printf("   Found Hostname: %s\n", settings.MainHostname)
			}
		}
		
		// Capture Relay IPs (generic)
		if matches := reRelay.FindStringSubmatch(line); len(matches) > 1 {
			if !strings.Contains(settings.MailWizzIP, matches[1]) {
				if settings.MailWizzIP != "" {
					settings.MailWizzIP += ", "
				}
				settings.MailWizzIP += matches[1]
			}
		}
	}

	// Auto-detect Main Server IP if not found
	if settings.MainServerIP == "" {
		settings.MainServerIP = getOutboundIP()
		fmt.Printf("   Auto-detected Server IP: %s\n", settings.MainServerIP)
	}

	// Upsert Settings
	existing, _ := st.GetSettings()
	if existing != nil {
		settings.ID = existing.ID
	}
	st.UpsertSettings(settings)
}

func parseSourcesToml(st *store.Store) {
	fmt.Println("🔍 Reading sources.toml for domains & IPs...")
	file, err := os.Open(SourcesPath)
	if err != nil {
		log.Fatalf("❌ Failed to open sources.toml: %v", err)
	}
	defer file.Close()

	reEhlo := regexp.MustCompile(`ehlo_domain\s*=\s*"([^"]+)"`)
	reSource := regexp.MustCompile(`source_address\s*=\s*"([^"]+)"`)

	var currentIP string
	count := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 1. Capture Source IP
		if matches := reSource.FindStringSubmatch(line); len(matches) > 1 {
			currentIP = matches[1]
			
			// FEATURE: Add to IP Inventory automatically
			if currentIP != "" {
				sysIP := &models.SystemIP{
					Value:     currentIP,
					CreatedAt: time.Now(),
					Interface: "eth0", // Default, can be edited in UI
				}
				st.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(sysIP)
			}
		}

		// 2. Capture Identity (EHLO)
		if matches := reEhlo.FindStringSubmatch(line); len(matches) > 1 {
			ehlo := matches[1] // e.g. "selector.example.com"
			
			// Generic splitting logic: assume generic format "selector.domain.tld"
			parts := strings.SplitN(ehlo, ".", 2)
			if len(parts) != 2 {
				continue
			}
			localPart := parts[0]
			domainName := parts[1]

			// A. Ensure Domain Exists
			domain := ensureDomain(st, domainName)

			// B. Create Standard Bounce User (b-localpart)
			bounceUser := "b-" + localPart
			ensureBounceAccount(st, bounceUser, domainName)

			// C. Create Sender
			sender := &models.Sender{
				DomainID:       domain.ID,
				LocalPart:      localPart,
				Email:          localPart + "@" + domainName,
				IP:             currentIP,
				BounceUsername: bounceUser,
			}
			
			// Upsert Sender
			var existing models.Sender
			if err := st.DB.Where("email = ?", sender.Email).First(&existing).Error; err == nil {
				existing.IP = currentIP
				existing.BounceUsername = bounceUser
				st.DB.Save(&existing)
			} else {
				st.CreateSender(sender)
				count++
			}
		}
	}
	fmt.Printf("   Imported %d senders.\n", count)
}

// Helpers

func ensureDomain(st *store.Store, name string) *models.Domain {
	d, err := st.GetDomainByName(name)
	if err == nil {
		return d
	}
	newD := &models.Domain{
		Name:            name,
		MailHost:        "mail." + name,
		BounceHost:      "bounce." + name,
		DMARCPolicy:     "none",
		DMARCPercentage: 100,
	}
	st.CreateDomain(newD)
	return newD
}

func ensureBounceAccount(st *store.Store, user, domain string) {
	var count int64
	st.DB.Model(&models.BounceAccount{}).Where("username = ?", user).Count(&count)
	if count == 0 {
		st.CreateBounceAccount(&models.BounceAccount{
			Username: user,
			Domain:   domain,
			Notes:    "Auto-imported",
		})
	}
}

// Auto-detect the primary outbound IP of the server
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
