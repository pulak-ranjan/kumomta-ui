package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	TOTPDigits   = 6
	TOTPPeriod   = 30 // seconds
	TOTPIssuer   = "KumoMTA-UI"
)

// GenerateTOTPSecret creates a new random secret for 2FA
func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateTOTPURI creates a URI for QR code generation
func GenerateTOTPURI(secret, email string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=%d&period=%d",
		TOTPIssuer, email, secret, TOTPIssuer, TOTPDigits, TOTPPeriod)
}

// ValidateTOTP checks if the provided code is valid
func ValidateTOTP(secret, code string) bool {
	// Allow for time drift - check current, previous, and next period
	now := time.Now().Unix()
	
	for _, offset := range []int64{-1, 0, 1} {
		timestamp := now + (offset * TOTPPeriod)
		expected := generateTOTPCode(secret, timestamp/TOTPPeriod)
		if expected == code {
			return true
		}
	}
	
	return false
}

// generateTOTPCode generates a TOTP code for a given counter
func generateTOTPCode(secret string, counter int64) string {
	// Decode secret
	secret = strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		// Try with padding
		for len(secret)%8 != 0 {
			secret += "="
		}
		key, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return ""
		}
	}

	// Convert counter to bytes (big-endian)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Modulo to get desired number of digits
	mod := uint32(1)
	for i := 0; i < TOTPDigits; i++ {
		mod *= 10
	}
	code = code % mod

	return fmt.Sprintf("%0*d", TOTPDigits, code)
}

// GetCurrentTOTP returns the current TOTP code (for testing)
func GetCurrentTOTP(secret string) string {
	counter := time.Now().Unix() / TOTPPeriod
	return generateTOTPCode(secret, counter)
}
