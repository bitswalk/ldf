package db

import (
	"database/sql"
	"fmt"
	"time"
)

// LanguagePackRepository handles language pack database operations
type LanguagePackRepository struct {
	db *Database
}

// NewLanguagePackRepository creates a new LanguagePackRepository
func NewLanguagePackRepository(db *Database) *LanguagePackRepository {
	return &LanguagePackRepository{db: db}
}

// List returns all language packs (metadata only, without full dictionary)
func (r *LanguagePackRepository) List() ([]LanguagePackMeta, error) {
	rows, err := r.db.DB().Query(`
		SELECT locale, name, version, author, created_at, updated_at
		FROM language_packs
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list language packs: %w", err)
	}
	defer rows.Close()

	var packs []LanguagePackMeta
	for rows.Next() {
		var pack LanguagePackMeta
		var author sql.NullString
		if err := rows.Scan(&pack.Locale, &pack.Name, &pack.Version, &author, &pack.CreatedAt, &pack.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan language pack: %w", err)
		}
		if author.Valid {
			pack.Author = author.String
		}
		packs = append(packs, pack)
	}

	return packs, rows.Err()
}

// Get retrieves a language pack by locale (with full dictionary)
func (r *LanguagePackRepository) Get(locale string) (*LanguagePack, error) {
	var pack LanguagePack
	var author sql.NullString

	err := r.db.DB().QueryRow(`
		SELECT locale, name, version, author, dictionary, created_at, updated_at
		FROM language_packs
		WHERE locale = ?
	`, locale).Scan(&pack.Locale, &pack.Name, &pack.Version, &author, &pack.Dictionary, &pack.CreatedAt, &pack.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get language pack: %w", err)
	}

	if author.Valid {
		pack.Author = author.String
	}

	return &pack, nil
}

// Create creates a new language pack
func (r *LanguagePackRepository) Create(pack *LanguagePack) error {
	now := time.Now().UTC()
	pack.CreatedAt = now
	pack.UpdatedAt = now

	var author interface{}
	if pack.Author != "" {
		author = pack.Author
	}

	_, err := r.db.DB().Exec(`
		INSERT INTO language_packs (locale, name, version, author, dictionary, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, pack.Locale, pack.Name, pack.Version, author, pack.Dictionary, pack.CreatedAt, pack.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create language pack: %w", err)
	}

	return nil
}

// Update updates an existing language pack
func (r *LanguagePackRepository) Update(pack *LanguagePack) error {
	pack.UpdatedAt = time.Now().UTC()

	var author interface{}
	if pack.Author != "" {
		author = pack.Author
	}

	result, err := r.db.DB().Exec(`
		UPDATE language_packs
		SET name = ?, version = ?, author = ?, dictionary = ?, updated_at = ?
		WHERE locale = ?
	`, pack.Name, pack.Version, author, pack.Dictionary, pack.UpdatedAt, pack.Locale)

	if err != nil {
		return fmt.Errorf("failed to update language pack: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("language pack not found: %s", pack.Locale)
	}

	return nil
}

// Delete removes a language pack by locale
func (r *LanguagePackRepository) Delete(locale string) error {
	result, err := r.db.DB().Exec(`DELETE FROM language_packs WHERE locale = ?`, locale)
	if err != nil {
		return fmt.Errorf("failed to delete language pack: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("language pack not found: %s", locale)
	}

	return nil
}

// Exists checks if a language pack exists for the given locale
func (r *LanguagePackRepository) Exists(locale string) (bool, error) {
	var count int
	err := r.db.DB().QueryRow(`SELECT COUNT(*) FROM language_packs WHERE locale = ?`, locale).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check language pack existence: %w", err)
	}
	return count > 0, nil
}
