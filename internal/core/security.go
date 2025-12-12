package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// Directory to store backups
	BackupDir = "/var/lib/kumomta-ui/backups"
)

// BlockIP adds an IP to the firewall drop zone immediately and permanently.
func BlockIP(ip string) error {
	// Security: Prevent blocking localhost or internal IPs by accident
	if ip == "127.0.0.1" || strings.HasPrefix(ip, "127.") {
		return fmt.Errorf("cannot block localhost")
	}
	if strings.Contains(ip, "/") || strings.Contains(ip, ";") || strings.Contains(ip, " ") {
		return fmt.Errorf("invalid IP format")
	}

	// 1. Immediate Block (Runtime)
	cmd := exec.Command("firewall-cmd", "--add-rich-rule", fmt.Sprintf("rule family='ipv4' source address='%s' drop", ip))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply runtime block: %v", err)
	}

	// 2. Permanent Block (Persist across reboots)
	cmdPerm := exec.Command("firewall-cmd", "--permanent", "--add-rich-rule", fmt.Sprintf("rule family='ipv4' source address='%s' drop", ip))
	if err := cmdPerm.Run(); err != nil {
		// Log but don't fail if runtime worked
		fmt.Printf("Warning: failed to make block permanent for %s: %v\n", ip, err)
	}

	return nil
}

// BackupConfig creates a timestamped tar.gz of the /opt/kumomta/etc directory.
func BackupConfig() error {
	if err := os.MkdirAll(BackupDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("config-backup-%s.tar.gz", timestamp)
	destPath := filepath.Join(BackupDir, filename)

	// Create archive
	// tar -czf /path/to/backup.tar.gz -C /opt/kumomta etc
	cmd := exec.Command("tar", "-czf", destPath, "-C", "/opt/kumomta", "etc")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("backup failed: %s", string(output))
	}

	// Retention Policy: Keep last 3 backups
	return pruneBackups()
}

func pruneBackups() error {
	entries, err := os.ReadDir(BackupDir)
	if err != nil {
		return err
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "config-backup-") {
			backups = append(backups, filepath.Join(BackupDir, e.Name()))
		}
	}

	// Sort (strings with timestamps sort correctly: old -> new)
	sort.Strings(backups)

	// If more than 3, delete the oldest ones
	if len(backups) > 3 {
		toDelete := backups[:len(backups)-3]
		for _, f := range toDelete {
			if err := os.Remove(f); err != nil {
				fmt.Printf("Warning: failed to prune old backup %s: %v\n", f, err)
			}
		}
	}
	return nil
}

var (
	keyOnce sync.Once
)

// Encryption Key Management
func getEncryptionKey() []byte {
	secret := os.Getenv("KUMO_APP_SECRET")
	if secret == "" {
		keyOnce.Do(func() {
			fmt.Println("WARNING: KUMO_APP_SECRET not set. Using insecure default key for encryption.")
		})
		// Fallback for development/quickstart
		// In production, this should be set!
		secret = "kumo-default-insecure-secret-key-32b"
	}
	// Pad or truncate to 32 bytes for AES-256
	b := []byte(secret)
	if len(b) < 32 {
		pad := make([]byte, 32-len(b))
		b = append(b, pad...)
	}
	return b[:32]
}

func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Backward compatibility: If it's not base64 or decryption fails, return original
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// Not base64, assume plaintext
		return ciphertext, nil
	}

	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		// Too short to be valid ciphertext, assume plaintext (if valid base64)
		return ciphertext, nil
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// Decryption failed (auth tag mismatch), so it wasn't encrypted with our key.
		// Assume legacy plaintext.
		return ciphertext, nil
	}

	return string(plaintext), nil
}
