package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	sm, err := NewSecretManager(keyPath)
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	plaintext := "my-secret-access-key-12345"
	encrypted, err := sm.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if !sm.IsEncrypted(encrypted) {
		t.Fatal("encrypted value should have encryption prefix")
	}

	if encrypted == plaintext {
		t.Fatal("encrypted value should differ from plaintext")
	}

	decrypted, err := sm.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted value = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	sm, err := NewSecretManager(keyPath)
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	encrypted, err := sm.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty string failed: %v", err)
	}
	if encrypted != "" {
		t.Fatalf("encrypting empty string should return empty, got %q", encrypted)
	}
}

func TestDecrypt_PlaintextPassthrough(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	sm, err := NewSecretManager(keyPath)
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	// Non-encrypted values should pass through unchanged
	plaintext := "not-encrypted-value"
	result, err := sm.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("Decrypt plaintext failed: %v", err)
	}
	if result != plaintext {
		t.Fatalf("plaintext passthrough: got %q, want %q", result, plaintext)
	}
}

func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	dir := t.TempDir()

	sm1, err := NewSecretManager(filepath.Join(dir, "key1.key"))
	if err != nil {
		t.Fatalf("NewSecretManager 1 failed: %v", err)
	}

	sm2, err := NewSecretManager(filepath.Join(dir, "key2.key"))
	if err != nil {
		t.Fatalf("NewSecretManager 2 failed: %v", err)
	}

	plaintext := "secret-data"
	encrypted, err := sm1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypting with a different key should fail
	_, err = sm2.Decrypt(encrypted)
	if err == nil {
		t.Fatal("decrypting with different key should fail")
	}
}

func TestIsEncrypted(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSecretManager(filepath.Join(dir, "test.key"))
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	if sm.IsEncrypted("plain-value") {
		t.Fatal("plain value should not be detected as encrypted")
	}
	if sm.IsEncrypted("") {
		t.Fatal("empty string should not be detected as encrypted")
	}
	if !sm.IsEncrypted("enc:v1:somedata") {
		t.Fatal("prefixed value should be detected as encrypted")
	}
}

func TestAutoKeyGeneration(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "subdir", "auto.key")

	sm, err := NewSecretManager(keyPath)
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	// Key file should have been created
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not created: %v", err)
	}
	if info.Size() != masterKeySize {
		t.Fatalf("key file size = %d, want %d", info.Size(), masterKeySize)
	}

	// Permissions should be restrictive
	if info.Mode().Perm() != 0600 {
		t.Fatalf("key file permissions = %o, want 0600", info.Mode().Perm())
	}

	// A second SecretManager with the same path should reuse the key
	sm2, err := NewSecretManager(keyPath)
	if err != nil {
		t.Fatalf("NewSecretManager (reuse) failed: %v", err)
	}

	// Both should encrypt/decrypt consistently
	encrypted, _ := sm.Encrypt("test")
	decrypted, err := sm2.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("cross-instance decrypt failed: %v", err)
	}
	if decrypted != "test" {
		t.Fatalf("cross-instance decrypt = %q, want %q", decrypted, "test")
	}
}
