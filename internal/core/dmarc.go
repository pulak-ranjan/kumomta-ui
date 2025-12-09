package core

import (
	"fmt"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// DMARCRecord represents a DMARC DNS record
type DMARCRecord struct {
	Domain   string `json:"domain"`
	DNSName  string `json:"dns_name"`
	DNSValue string `json:"dns_value"`
	Policy   string `json:"policy"`
}

// GenerateDMARCRecord creates a DMARC record for a domain
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

// GenerateAllDNSRecords generates all DNS records for a domain
type AllDNSRecords struct {
	Domain    string        `json:"domain"`
	A         []DNSRecord   `json:"a"`
	MX        []DNSRecord   `json:"mx"`
	SPF       DNSRecord     `json:"spf"`
	DMARC     DNSRecord     `json:"dmarc"`
	DKIM      []DKIMDNSRecord `json:"dkim"`
}

type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

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

// ValidateDMARCPolicy checks if policy is valid
func ValidateDMARCPolicy(policy string) bool {
	switch policy {
	case "none", "quarantine", "reject":
		return true
	default:
		return false
	}
}
