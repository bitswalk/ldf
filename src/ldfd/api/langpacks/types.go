package langpacks

import (
	"encoding/json"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles language pack HTTP requests
type Handler struct {
	langPackRepo *db.LanguagePackRepository
}

// Config contains configuration options for the Handler
type Config struct {
	LangPackRepo *db.LanguagePackRepository
}

// LanguagePackMeta represents the metadata from a language pack's meta.json
type LanguagePackMeta struct {
	Locale  string `json:"locale"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  string `json:"author,omitempty"`
}

// LanguagePackListResponse represents the response for listing language packs
type LanguagePackListResponse struct {
	LanguagePacks []db.LanguagePackMeta `json:"language_packs"`
}

// LanguagePackResponse represents a single language pack response
type LanguagePackResponse struct {
	Locale     string          `json:"locale"`
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Author     string          `json:"author,omitempty"`
	Dictionary json.RawMessage `json:"dictionary"`
}
