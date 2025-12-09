package core

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

const (
	MaildirBase = "/home"
)

// CreateBounceAccount creates a system user for bounce handling
func CreateBounceAccount(username, domain string, st *store.Store) error {
	// Generate random password
	password := generateRandomPassword(16)
	return CreateBounceAccountWithPassword(username, domain, password, "", st)
}

func CreateBounceAccountWithPassword(username, domain, password, notes string, st *store.Store) error {
	// Check if user exists
	cmd := exec.Command("id", username)
	if cmd.Run() == nil {
		// User already exists
		return nil
	}

	// Create system user
	cmd = exec.Command("useradd", "-m", "-s", "/usr/sbin/nologin", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	// Set password
	if password == "" {
		password = generateRandomPassword(16)
	}
	cmd = exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(username + ":" + password)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password: %v", err)
	}

	// Create Maildir
	maildir := filepath.Join(MaildirBase, username, "Maildir")
	os.MkdirAll(filepath.Join(maildir, "cur"), 0700)
	os.MkdirAll(filepath.Join(maildir, "new"), 0700)
	os.MkdirAll(filepath.Join(maildir, "tmp"), 0700)

	// Set ownership
	exec.Command("chown", "-R", username+":"+username, filepath.Join(MaildirBase, username)).Run()

	// Hash password for DB
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Save to DB
	bounce := &models.BounceAccount{
		Username:     username,
		PasswordHash: string(hash),
		Domain:       domain,
		Notes:        notes,
	}

	return st.CreateBounceAccount(bounce)
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// IP Utilities

type DetectedIP struct {
	IP        string `json:"ip"`
	Interface string `json:"interface"`
}

// DetectServerIPs finds all IPv4 addresses on the server
func DetectServerIPs() []DetectedIP {
	var detected []DetectedIP

	ifaces, err := net.Interfaces()
	if err != nil {
		return detected
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue // Skip IPv6
			}

			// Skip private ranges often used for internal networking
			if ip[0] == 127 {
				continue
			}

			detected = append(detected, DetectedIP{
				IP:        ip.String(),
				Interface: iface.Name,
			})
		}
	}

	return detected
}

// ExpandCIDR expands a CIDR notation to list of IPs
func ExpandCIDR(cidr string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %v", err)
	}

	var ips []string
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); incrementIP(ip) {
		// Skip network and broadcast addresses for /24 and smaller
		ones, bits := ipNet.Mask.Size()
		if ones >= 24 {
			if ip[3] == 0 || ip[3] == 255 {
				continue
			}
		}

		ips = append(ips, ip.String())

		// Safety limit
		if len(ips) > 1000 {
			break
		}
	}

	// Handle single IP
	if bits-ones == 0 {
		ips = []string{ipNet.IP.String()}
	}

	return ips, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
