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
//
// @Summary      List language packs
// @Description  Returns all available custom language packs with metadata
// @Tags         LanguagePacks
// @Produce      json
// @Success      200   {object}  LanguagePackListResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/language-packs [get]
func (h *Handler) HandleList(c *gin.Context) {
	packs, err := h.langPackRepo.List()
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to list language packs: %v", err))
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
//
// @Summary      Get language pack
// @Description  Returns a specific language pack including its full translation dictionary
// @Tags         LanguagePacks
// @Produce      json
// @Param        locale  path      string  true  "Locale identifier (e.g. fr, de, ja)"
// @Success      200     {object}  LanguagePackResponse
// @Failure      401     {object}  common.ErrorResponse
// @Failure      404     {object}  common.ErrorResponse
// @Failure      500     {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/language-packs/{locale} [get]
func (h *Handler) HandleGet(c *gin.Context) {
	locale := c.Param("locale")

	pack, err := h.langPackRepo.Get(locale)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to get language pack: %v", err))
		return
	}

	if pack == nil {
		common.NotFound(c, fmt.Sprintf("Language pack '%s' not found", locale))
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
//
// @Summary      Upload language pack
// @Description  Uploads a new language pack from a .tar.xz, .tar.gz, or .xz archive containing meta.json and translation files
// @Tags         LanguagePacks
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Language pack archive (.tar.xz, .tar.gz, or .xz)"
// @Success      200   {object}  object  "Language pack updated"
// @Success      201   {object}  object  "Language pack created"
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/language-packs [post]
func (h *Handler) HandleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.BadRequest(c, "No file provided. Please upload a .tar.xz, .tar.gz, or .xz archive.")
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
		common.BadRequest(c, "Unsupported archive format. Please use .tar.xz, .tar.gz, or .xz")
		return
	}

	if err != nil {
		common.BadRequest(c, fmt.Sprintf("Failed to extract archive: %v", err))
		return
	}

	if meta == nil {
		common.BadRequest(c, "Archive must contain a meta.json file with locale, name, and version fields")
		return
	}

	if meta.Locale == "" || meta.Name == "" || meta.Version == "" {
		common.BadRequest(c, "meta.json must contain locale, name, and version fields")
		return
	}

	dictJSON, err := json.Marshal(dictionary)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to serialize dictionary: %v", err))
		return
	}

	exists, err := h.langPackRepo.Exists(meta.Locale)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to check language pack existence: %v", err))
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
			common.InternalError(c, fmt.Sprintf("Failed to update language pack: %v", err))
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
			common.InternalError(c, fmt.Sprintf("Failed to create language pack: %v", err))
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
//
// @Summary      Delete language pack
// @Description  Deletes a language pack by its locale identifier
// @Tags         LanguagePacks
// @Produce      json
// @Param        locale  path      string  true  "Locale identifier (e.g. fr, de, ja)"
// @Success      200     {object}  object
// @Failure      401     {object}  common.ErrorResponse
// @Failure      403     {object}  common.ErrorResponse
// @Failure      404     {object}  common.ErrorResponse
// @Failure      500     {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/language-packs/{locale} [delete]
func (h *Handler) HandleDelete(c *gin.Context) {
	locale := c.Param("locale")

	exists, err := h.langPackRepo.Exists(locale)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to check language pack: %v", err))
		return
	}

	if !exists {
		common.NotFound(c, fmt.Sprintf("Language pack '%s' not found", locale))
		return
	}

	if err := h.langPackRepo.Delete(locale); err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to delete language pack: %v", err))
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
