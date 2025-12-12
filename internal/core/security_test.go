package core

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// 1. Test with default key (no env var)
	original := "my-secret-api-key-123"

	encrypted, err := Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == original {
		t.Fatal("Encrypted string is same as original")
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("Expected %s, got %s", original, decrypted)
	}

	// 2. Test with custom key (env var)
	os.Setenv("KUMO_APP_SECRET", "custom-super-secret-key-that-is-very-long-32b")
	defer os.Unsetenv("KUMO_APP_SECRET")

	encrypted2, err := Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt with custom key failed: %v", err)
	}

	if encrypted2 == encrypted {
		t.Fatal("Encryption should differ with different keys (and nonces)")
	}

	decrypted2, err := Decrypt(encrypted2)
	if err != nil {
		t.Fatalf("Decrypt with custom key failed: %v", err)
	}

	if decrypted2 != original {
		t.Errorf("Expected %s, got %s", original, decrypted2)
	}
}

func TestEncryptEmpty(t *testing.T) {
	enc, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}
	if enc != "" {
		t.Errorf("Expected empty string, got %s", enc)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Plaintext that is not valid base64
	plaintext1 := "my-plaintext-key"
	dec1, err := Decrypt(plaintext1)
	if err != nil {
		t.Fatalf("Decrypt plaintext1 failed: %v", err)
	}
	if dec1 != plaintext1 {
		t.Errorf("Expected %s, got %s", plaintext1, dec1)
	}

	// Plaintext that MIGHT be base64 (if we force it), but won't decrypt with our key
	// "Hello world" in base64 is "SGVsbG8gd29ybGQ="
	plaintext2 := "SGVsbG8gd29ybGQ="
	dec2, err := Decrypt(plaintext2)
	if err != nil {
		t.Fatalf("Decrypt plaintext2 failed: %v", err)
	}
	// It should fail AES decryption (auth tag mismatch or block size) and fallback to original
	if dec2 != plaintext2 {
		t.Errorf("Expected fallback to %s, got %s", plaintext2, dec2)
	}
}
