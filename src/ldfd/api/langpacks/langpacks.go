package langpacks

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
	"github.com/ulikunitz/xz"
)

// NewHandler creates a new langpacks handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		langPackRepo: cfg.LangPackRepo,
	}
}

// HandleList returns all available custom language packs
func (h *Handler) HandleList(c *gin.Context) {
	packs, err := h.langPackRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to list language packs: %v", err),
		})
		return
	}

	if packs == nil {
		packs = []db.LanguagePackMeta{}
	}

	c.JSON(http.StatusOK, LanguagePackListResponse{
		LanguagePacks: packs,
	})
}

// HandleGet returns a specific language pack by locale
func (h *Handler) HandleGet(c *gin.Context) {
	locale := c.Param("locale")

	pack, err := h.langPackRepo.Get(locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to get language pack: %v", err),
		})
		return
	}

	if pack == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Language pack '%s' not found", locale),
		})
		return
	}

	c.JSON(http.StatusOK, LanguagePackResponse{
		Locale:     pack.Locale,
		Name:       pack.Name,
		Version:    pack.Version,
		Author:     pack.Author,
		Dictionary: json.RawMessage(pack.Dictionary),
	})
}

// HandleUpload handles uploading a new language pack archive
func (h *Handler) HandleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "No file provided. Please upload a .tar.xz, .tar.gz, or .xz archive.",
		})
		return
	}
	defer file.Close()

	filename := header.Filename

	var meta *LanguagePackMeta
	var dictionary map[string]interface{}

	if strings.HasSuffix(filename, ".tar.xz") {
		meta, dictionary, err = extractTarXZ(file)
	} else if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		meta, dictionary, err = extractTarGZ(file)
	} else if strings.HasSuffix(filename, ".xz") {
		meta, dictionary, err = extractXZ(file)
	} else {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Unsupported archive format. Please use .tar.xz, .tar.gz, or .xz",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Failed to extract archive: %v", err),
		})
		return
	}

	if meta == nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Archive must contain a meta.json file with locale, name, and version fields",
		})
		return
	}

	if meta.Locale == "" || meta.Name == "" || meta.Version == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "meta.json must contain locale, name, and version fields",
		})
		return
	}

	dictJSON, err := json.Marshal(dictionary)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to serialize dictionary: %v", err),
		})
		return
	}

	exists, err := h.langPackRepo.Exists(meta.Locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to check language pack existence: %v", err),
		})
		return
	}

	pack := &db.LanguagePack{
		Locale:     meta.Locale,
		Name:       meta.Name,
		Version:    meta.Version,
		Author:     meta.Author,
		Dictionary: string(dictJSON),
	}

	if exists {
		if err := h.langPackRepo.Update(pack); err != nil {
			c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Error:   "Internal server error",
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("Failed to update language pack: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Language pack updated successfully",
			"locale":  meta.Locale,
			"name":    meta.Name,
			"version": meta.Version,
		})
	} else {
		if err := h.langPackRepo.Create(pack); err != nil {
			c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Error:   "Internal server error",
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("Failed to create language pack: %v", err),
			})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": "Language pack created successfully",
			"locale":  meta.Locale,
			"name":    meta.Name,
			"version": meta.Version,
		})
	}
}

// HandleDelete deletes a language pack by locale
func (h *Handler) HandleDelete(c *gin.Context) {
	locale := c.Param("locale")

	exists, err := h.langPackRepo.Exists(locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to check language pack: %v", err),
		})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Language pack '%s' not found", locale),
		})
		return
	}

	if err := h.langPackRepo.Delete(locale); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to delete language pack: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Language pack deleted successfully",
		"locale":  locale,
	})
}

// extractTarXZ extracts a .tar.xz archive and returns meta and merged dictionary
func extractTarXZ(r io.Reader) (*LanguagePackMeta, map[string]interface{}, error) {
	xzReader, err := xz.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create xz reader: %w", err)
	}

	return extractTar(xzReader)
}

// extractTarGZ extracts a .tar.gz archive and returns meta and merged dictionary
func extractTarGZ(r io.Reader) (*LanguagePackMeta, map[string]interface{}, error) {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return extractTar(gzReader)
}

// extractTar extracts a tar archive and returns meta and merged dictionary
func extractTar(r io.Reader) (*LanguagePackMeta, map[string]interface{}, error) {
	tarReader := tar.NewReader(r)

	var meta *LanguagePackMeta
	dictionary := make(map[string]interface{})

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		name := header.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}

		if !strings.HasSuffix(name, ".json") {
			continue
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}

		if name == "meta.json" {
			var m LanguagePackMeta
			if err := json.Unmarshal(content, &m); err != nil {
				return nil, nil, fmt.Errorf("failed to parse meta.json: %w", err)
			}
			meta = &m
		} else {
			var data map[string]interface{}
			if err := json.Unmarshal(content, &data); err != nil {
				return nil, nil, fmt.Errorf("failed to parse %s: %w", name, err)
			}
			namespace := strings.TrimSuffix(name, ".json")
			dictionary[namespace] = data
		}
	}

	return meta, dictionary, nil
}

// extractXZ extracts a single .xz compressed JSON file
func extractXZ(r io.Reader) (*LanguagePackMeta, map[string]interface{}, error) {
	xzReader, err := xz.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create xz reader: %w", err)
	}

	content, err := io.ReadAll(xzReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read xz content: %w", err)
	}

	var combined struct {
		Meta       LanguagePackMeta       `json:"meta"`
		Dictionary map[string]interface{} `json:"dictionary"`
	}

	if err := json.Unmarshal(content, &combined); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if combined.Meta.Locale == "" {
		return nil, nil, fmt.Errorf("missing meta.locale in JSON")
	}

	return &combined.Meta, combined.Dictionary, nil
}
