package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// EnsureBounceAccount makes sure a system user exists for the given bounce account.
// If plainPassword is non-empty, it sets/updates the system password.
// It always ensures Maildir structure exists.
func EnsureBounceAccount(acc models.BounceAccount, plainPassword string) error {
	if acc.Username == "" {
		return fmt.Errorf("username is required")
	}

	// FIX: Validate username format to prevent shell injection
	// Only allow lowercase alphanumeric, underscores, and hyphens.
	validUsername := regexp.MustCompile(`^[a-z0-9_-]+$`)
	if !validUsername.MatchString(acc.Username) {
		return fmt.Errorf("invalid username format: must be [a-z0-9_-]")
	}

	// Check if user exists: id -u username
	checkCmd := exec.Command("id", "-u", acc.Username)
	if err := checkCmd.Run(); err != nil {
		// User does not exist, create it
		// Safe to pass acc.Username because of regex check above
		useradd := exec.Command("useradd", "-m", "-s", "/usr/sbin/nologin", acc.Username)
		if out, err := useradd.CombinedOutput(); err != nil {
			return fmt.Errorf("useradd failed: %v, output: %s", err, string(out))
		}
	}

	// Set password via chpasswd only if provided
	if plainPassword != "" {
		chpasswd := exec.Command("chpasswd")
		chpasswd.Stdin = bytes.NewBufferString(fmt.Sprintf("%s:%s\n", acc.Username, plainPassword))
		if out, err := chpasswd.CombinedOutput(); err != nil {
			return fmt.Errorf("chpasswd failed: %v, output: %s", err, string(out))
		}
	}

	// Ensure Maildir exists
	homeDir := filepath.Join("/home", acc.Username)
	maildir := filepath.Join(homeDir, "Maildir")
	subdirs := []string{
		filepath.Join(maildir, "cur"),
		filepath.Join(maildir, "new"),
		filepath.Join(maildir, "tmp"),
	}
	for _, d := range subdirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("mkdir Maildir: %w", err)
		}
	}

	// chown -R username:username /home/username/Maildir
	// Use explicit user:group string format
	chown := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", acc.Username, acc.Username), maildir)
	if out, err := chown.CombinedOutput(); err != nil {
		return fmt.Errorf("chown Maildir failed: %v, output: %s", err, string(out))
	}

	return nil
}

// ApplyAllBounceAccounts ensures all stored bounce accounts exist on system.
// It does NOT change passwords (plain password is not stored).
func ApplyAllBounceAccounts(accounts []models.BounceAccount) error {
	for _, acc := range accounts {
		if err := EnsureBounceAccount(acc, ""); err != nil {
			return err
		}
	}
	return nil
}
