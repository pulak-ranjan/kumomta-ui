package core

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// DMARCRecord represents a DMARC DNS record
type DMARCRecord struct {
	Domain   string `json:"domain"`
	DNSName  string `json:"dns_name"`
	DNSValue string `json:"dns_value"`
	Policy   string `json:"policy"`
}

// AllDNSRecords holds both generated and live DNS states
type AllDNSRecords struct {
	Domain string          `json:"domain"`
	A      []DNSRecord     `json:"a"`
	MX     []DNSRecord     `json:"mx"`
	SPF    DNSRecord       `json:"spf"`
	DMARC  DNSRecord       `json:"dmarc"`
	DKIM   []DKIMDNSRecord `json:"dkim"`
}

type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// GenerateDMARCRecord creates a DMARC record for a domain based on DB settings
func GenerateDMARCRecord(domain *models.Domain) DMARCRecord {
	policy := domain.DMARCPolicy
	if policy == "" {
		policy = "none" // Safe default
	}

	// Build DMARC value
	parts := []string{
		"v=DMARC1",
		fmt.Sprintf("p=%s", policy),
	}

	// Add percentage if not 100%
	if domain.DMARCPercentage > 0 && domain.DMARCPercentage < 100 {
		parts = append(parts, fmt.Sprintf("pct=%d", domain.DMARCPercentage))
	}

	// Add aggregate report address
	if domain.DMARCRua != "" {
		parts = append(parts, fmt.Sprintf("rua=mailto:%s", domain.DMARCRua))
	}

	// Add forensic report address
	if domain.DMARCRuf != "" {
		parts = append(parts, fmt.Sprintf("ruf=mailto:%s", domain.DMARCRuf))
	}

	// Additional recommended settings
	parts = append(parts, "adkim=r") // Relaxed DKIM alignment
	parts = append(parts, "aspf=r")  // Relaxed SPF alignment

	return DMARCRecord{
		Domain:   domain.Name,
		DNSName:  fmt.Sprintf("_dmarc.%s", domain.Name),
		DNSValue: strings.Join(parts, "; "),
		Policy:   policy,
	}
}

// GenerateAllDNSRecords generates expected DNS records based on configuration
func GenerateAllDNSRecords(domain *models.Domain, mainIP string, snap *Snapshot) AllDNSRecords {
	records := AllDNSRecords{
		Domain: domain.Name,
	}

	mailHost := domain.MailHost
	if mailHost == "" {
		mailHost = "mail." + domain.Name
	}

	bounceHost := domain.BounceHost
	if bounceHost == "" {
		bounceHost = "bounce." + domain.Name
	}

	// A Records
	records.A = []DNSRecord{
		{Name: mailHost, Type: "A", Value: mainIP, TTL: 3600},
		{Name: bounceHost, Type: "A", Value: mainIP, TTL: 3600},
	}

	// MX Record
	records.MX = []DNSRecord{
		{Name: domain.Name, Type: "MX", Value: fmt.Sprintf("10 %s.", mailHost), TTL: 3600},
	}

	// SPF Record - collect all IPs
	ips := make(map[string]bool)
	ips[mainIP] = true
	for _, sender := range domain.Senders {
		if sender.IP != "" {
			ips[sender.IP] = true
		}
	}

	ipParts := []string{}
	for ip := range ips {
		ipParts = append(ipParts, fmt.Sprintf("ip4:%s", ip))
	}

	spfValue := fmt.Sprintf("v=spf1 %s ~all", strings.Join(ipParts, " "))
	records.SPF = DNSRecord{
		Name:  domain.Name,
		Type:  "TXT",
		Value: spfValue,
		TTL:   3600,
	}

	// DMARC Record
	dmarc := GenerateDMARCRecord(domain)
	records.DMARC = DNSRecord{
		Name:  dmarc.DNSName,
		Type:  "TXT",
		Value: dmarc.DNSValue,
		TTL:   3600,
	}

	// DKIM Records (from existing function)
	if snap != nil {
		dkimRecs, _ := ListDKIMDNSRecords(snap)
		for _, rec := range dkimRecs {
			if rec.Domain == domain.Name {
				records.DKIM = append(records.DKIM, rec)
			}
		}
	}

	return records
}

// LookupLiveDNS queries the actual DNS records for the domain
func LookupLiveDNS(domain *models.Domain) (AllDNSRecords, error) {
	records := AllDNSRecords{
		Domain: domain.Name,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Helper to add A records
	addA := func(name string) {
		defer wg.Done()
		ips, err := net.LookupIP(name)
		if err == nil {
			for _, ip := range ips {
				if ipv4 := ip.To4(); ipv4 != nil {
					mu.Lock()
					records.A = append(records.A, DNSRecord{Name: name, Type: "A", Value: ipv4.String()})
					mu.Unlock()
				}
			}
		}
	}

	// Helper to add MX records
	addMX := func() {
		defer wg.Done()
		mxs, err := net.LookupMX(domain.Name)
		if err == nil {
			for _, mx := range mxs {
				mu.Lock()
				records.MX = append(records.MX, DNSRecord{Name: domain.Name, Type: "MX", Value: fmt.Sprintf("%d %s", mx.Pref, mx.Host)})
				mu.Unlock()
			}
		}
	}

	// Helper to add TXT records (SPF, DMARC, DKIM)
	addTXT := func() {
		defer wg.Done()
		// Root TXT (SPF)
		txts, _ := net.LookupTXT(domain.Name)
		for _, txt := range txts {
			if strings.HasPrefix(txt, "v=spf1") {
				mu.Lock()
				records.SPF = DNSRecord{Name: domain.Name, Type: "TXT", Value: txt}
				mu.Unlock()
			}
		}

		// DMARC
		dmarcs, _ := net.LookupTXT("_dmarc." + domain.Name)
		for _, txt := range dmarcs {
			if strings.HasPrefix(txt, "v=DMARC1") {
				mu.Lock()
				records.DMARC = DNSRecord{Name: "_dmarc." + domain.Name, Type: "TXT", Value: txt}
				mu.Unlock()
			}
		}

		// DKIM (Check all senders)
		// We use a map to avoid checking the same selector twice
		checkedSelectors := make(map[string]bool)
		for _, s := range domain.Senders {
			if s.LocalPart == "" || checkedSelectors[s.LocalPart] {
				continue
			}
			checkedSelectors[s.LocalPart] = true
			
			dkimName := s.LocalPart + "._domainkey." + domain.Name
			dkimTxts, _ := net.LookupTXT(dkimName)
			for _, txt := range dkimTxts {
				if strings.HasPrefix(txt, "v=DKIM1") {
					mu.Lock()
					records.DKIM = append(records.DKIM, DKIMDNSRecord{
						Domain:   domain.Name,
						Selector: s.LocalPart,
						DNSName:  dkimName,
						DNSValue: txt,
					})
					mu.Unlock()
				}
			}
		}
	}

	wg.Add(3)
	// 1. A Records
	go func() {
		defer wg.Done()
		var subWg sync.WaitGroup
		
		mailHost := domain.MailHost
		if mailHost == "" { mailHost = "mail." + domain.Name }
		
		bounceHost := domain.BounceHost
		if bounceHost == "" { bounceHost = "bounce." + domain.Name }

		subWg.Add(2)
		go addA(mailHost)
		go addA(bounceHost)
		subWg.Wait()
	}()

	// 2. MX Records
	go addMX()

	// 3. TXT Records (SPF/DMARC/DKIM)
	go addTXT()

	wg.Wait()
	return records, nil
}

// ValidateDMARCPolicy checks if policy is valid
func ValidateDMARCPolicy(policy string) bool {
	switch policy {
	case "none", "quarantine", "reject":
		return true
	default:
		return false
	}
}
