// Package security provides cryptographic utilities for secrets management.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// encryptedPrefix marks encrypted values in the database.
	encryptedPrefix = "enc:v1:"
	// masterKeySize is the AES-256 key size in bytes.
	masterKeySize = 32
)

// SecretManager handles encryption and decryption of sensitive settings.
type SecretManager struct {
	masterKey []byte
}

// NewSecretManager creates a SecretManager from a key file path.
// If the key file does not exist, a new random key is generated and saved.
func NewSecretManager(keyPath string) (*SecretManager, error) {
	keyPath = expandHome(keyPath)

	key, err := os.ReadFile(keyPath)
	if err == nil && len(key) == masterKeySize {
		return &SecretManager{masterKey: key}, nil
	}

	// Generate a new master key
	key = make([]byte, masterKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write key file with restrictive permissions
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to write master key: %w", err)
	}

	return &SecretManager{masterKey: key}, nil
}

// Encrypt encrypts a plaintext string using AES-256-GCM.
// Returns a string with the "enc:v1:" prefix followed by base64-encoded ciphertext.
func (sm *SecretManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(sm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// nonce is prepended to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encryptedPrefix + encoded, nil
}

// Decrypt decrypts a value. If the value doesn't have the encryption prefix,
// it is returned as-is (backward compatibility with existing plaintext values).
func (sm *SecretManager) Decrypt(value string) (string, error) {
	if !sm.IsEncrypted(value) {
		return value, nil
	}

	encoded := strings.TrimPrefix(value, encryptedPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(sm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted returns true if the value has the encryption prefix.
func (sm *SecretManager) IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
