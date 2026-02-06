package downloads

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// MirrorHandler handles mirror configuration API endpoints
type MirrorHandler struct {
	mirrorRepo *db.MirrorConfigRepository
}

// NewMirrorHandler creates a new mirror handler
func NewMirrorHandler(mirrorRepo *db.MirrorConfigRepository) *MirrorHandler {
	return &MirrorHandler{mirrorRepo: mirrorRepo}
}

// HandleListMirrors lists all configured mirrors
// @Summary      List mirrors
// @Description  Lists all configured download mirrors
// @Tags         Mirrors
// @Produce      json
// @Success      200  {object}  MirrorListResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/mirrors [get]
func (h *MirrorHandler) HandleListMirrors(c *gin.Context) {
	mirrors, err := h.mirrorRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	response := make([]MirrorResponse, 0, len(mirrors))
	for _, m := range mirrors {
		response = append(response, mirrorToResponse(m))
	}

	c.JSON(http.StatusOK, MirrorListResponse{
		Count:   len(response),
		Mirrors: response,
	})
}

// HandleCreateMirror creates a new mirror configuration
// @Summary      Create mirror
// @Description  Creates a new download mirror configuration
// @Tags         Mirrors
// @Accept       json
// @Produce      json
// @Param        body  body      CreateMirrorRequest  true  "Mirror configuration"
// @Success      201   {object}  MirrorResponse
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/mirrors [post]
func (h *MirrorHandler) HandleCreateMirror(c *gin.Context) {
	var req CreateMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Name == "" || req.URLPrefix == "" || req.MirrorURL == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "name, url_prefix, and mirror_url are required",
		})
		return
	}

	entry := &db.MirrorConfigEntry{
		Name:      req.Name,
		URLPrefix: req.URLPrefix,
		MirrorURL: req.MirrorURL,
		Priority:  req.Priority,
		Enabled:   true,
	}
	if req.Enabled != nil {
		entry.Enabled = *req.Enabled
	}

	if err := h.mirrorRepo.Create(entry); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, mirrorToResponse(*entry))
}

// HandleUpdateMirror updates an existing mirror configuration
// @Summary      Update mirror
// @Description  Updates an existing download mirror configuration
// @Tags         Mirrors
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "Mirror ID"
// @Param        body  body      UpdateMirrorRequest   true  "Mirror configuration"
// @Success      200   {object}  MirrorResponse
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/mirrors/{id} [put]
func (h *MirrorHandler) HandleUpdateMirror(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Mirror ID required",
		})
		return
	}

	existing, err := h.mirrorRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Mirror not found",
		})
		return
	}

	var req UpdateMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.URLPrefix != "" {
		existing.URLPrefix = req.URLPrefix
	}
	if req.MirrorURL != "" {
		existing.MirrorURL = req.MirrorURL
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.mirrorRepo.Update(existing); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, mirrorToResponse(*existing))
}

// HandleDeleteMirror deletes a mirror configuration
// @Summary      Delete mirror
// @Description  Deletes a download mirror configuration
// @Tags         Mirrors
// @Param        id   path      string  true  "Mirror ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/mirrors/{id} [delete]
func (h *MirrorHandler) HandleDeleteMirror(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Mirror ID required",
		})
		return
	}

	if err := h.mirrorRepo.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Mirror not found",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// mirrorToResponse converts a db entry to an API response
func mirrorToResponse(m db.MirrorConfigEntry) MirrorResponse {
	return MirrorResponse{
		ID:        m.ID,
		Name:      m.Name,
		URLPrefix: m.URLPrefix,
		MirrorURL: m.MirrorURL,
		Priority:  m.Priority,
		Enabled:   m.Enabled,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
