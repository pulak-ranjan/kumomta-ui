package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
