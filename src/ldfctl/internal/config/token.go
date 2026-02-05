package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitswalk/ldf/src/common/paths"
)

const tokenFileName = "token.json"

// TokenData holds the stored authentication tokens
type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	ServerURL    string `json:"server_url"`
	Username     string `json:"username"`
}

func tokenFilePath() string {
	return paths.Expand("~/.ldfctl/" + tokenFileName)
}

// SaveToken writes the token data to disk
func SaveToken(data *TokenData) error {
	path := tokenFilePath()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken reads the token data from disk
func LoadToken() (*TokenData, error) {
	path := tokenFilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &tokenData, nil
}

// ClearToken removes the token file from disk
func ClearToken() error {
	path := tokenFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	return nil
}
