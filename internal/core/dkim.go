package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// Base path for DKIM keys on disk.
// Private keys: /opt/kumomta/etc/dkim/<domain>/<selector>.key
// Public keys:  /opt/kumomta/etc/dkim/<domain>/<selector>.pub
const DKIMBasePath = "/opt/kumomta/etc/dkim"

func dkimKeyPaths(domain, selector string) (privPath, pubPath, dir string) {
	dir = filepath.Join(DKIMBasePath, domain)
	privPath = filepath.Join(dir, selector+".key")
	pubPath = filepath.Join(dir, selector+".pub")
	return
}

// GenerateDKIMKey creates a new RSA keypair for domain+selector
// and writes them to DKIMBasePath/domain/(selector.key|selector.pub).
func GenerateDKIMKey(domain, selector string) error {
	privPath, pubPath, dir := dkimKeyPaths(domain, selector)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir dkim dir: %w", err)
	}

	// Generate RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate rsa key: %w", err)
	}

	// Encode private key in PEM (PKCS1)
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	// Encode public key in PEM (PKIX)
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// Write private key
	if err := os.WriteFile(privPath, privPem, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	// Write public key
	if err := os.WriteFile(pubPath, pubPem, 0o644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}

// DKIMDNSRecord represents a single DNS TXT record for DKIM.
type DKIMDNSRecord struct {
	Domain   string `json:"domain"`
	Selector string `json:"selector"`
	DNSName  string `json:"dns_name"`  // selector._domainkey.domain
	DNSValue string `json:"dns_value"` // v=DKIM1; k=rsa; p=...
}

// ListDKIMDNSRecords iterates over all domains+senders, reads .pub files,
// and constructs ready-to-paste DNS records.
func ListDKIMDNSRecords(snap *Snapshot) ([]DKIMDNSRecord, error) {
	var records []DKIMDNSRecord

	for _, d := range snap.Domains {
		for _, s := range d.Senders {
			selector := s.LocalPart
			if selector == "" {
				continue
			}

			_, pubPath, _ := dkimKeyPaths(d.Name, selector)
			data, err := os.ReadFile(pubPath)
			if err != nil {
				// If file doesn't exist, just skip this one
				continue
			}

			pubBase64 := extractPEMBase64(string(data))
			if pubBase64 == "" {
				continue
			}

			name := fmt.Sprintf("%s._domainkey.%s", selector, d.Name)
			value := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", pubBase64)

			records = append(records, DKIMDNSRecord{
				Domain:   d.Name,
				Selector: selector,
				DNSName:  name,
				DNSValue: value,
			})
		}
	}

	return records, nil
}

// extractPEMBase64 strips PEM header/footer and newlines to get pure base64.
func extractPEMBase64(pemStr string) string {
	lines := strings.Split(pemStr, "\n")
	var b strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-----") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}

// GenerateDKIMForDomainAllSenders generates keys for all senders of a domain.
func GenerateDKIMForDomainAllSenders(domain models.Domain) error {
	for _, s := range domain.Senders {
		if s.LocalPart == "" {
			continue
		}
		if err := GenerateDKIMKey(domain.Name, s.LocalPart); err != nil {
			return fmt.Errorf("generate dkim for %s/%s: %w", domain.Name, s.LocalPart, err)
		}
	}
	return nil
}
