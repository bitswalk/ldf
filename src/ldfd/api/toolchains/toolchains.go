package toolchains

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new toolchain profiles handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		repo: cfg.ToolchainProfileRepo,
	}
}

// HandleList returns all toolchain profiles, optionally filtered by type
// @Summary      List toolchain profiles
// @Description  Returns all toolchain profiles, optionally filtered by type (gcc, llvm)
// @Tags         Toolchain Profiles
// @Produce      json
// @Param        type  query     string  false  "Filter by toolchain type (gcc, llvm)"
// @Success      200   {object}  ToolchainProfileListResponse
// @Failure      500   {object}  common.ErrorResponse
// @Router       /v1/toolchains [get]
func (h *Handler) HandleList(c *gin.Context) {
	var profiles []db.ToolchainProfile
	var err error

	if toolchainType := c.Query("type"); toolchainType != "" {
		profiles, err = h.repo.ListByType(toolchainType)
	} else {
		profiles, err = h.repo.List()
	}

	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if profiles == nil {
		profiles = []db.ToolchainProfile{}
	}

	c.JSON(http.StatusOK, ToolchainProfileListResponse{
		Count:    len(profiles),
		Profiles: profiles,
	})
}

// HandleGet returns a single toolchain profile by ID
// @Summary      Get a toolchain profile
// @Description  Returns a single toolchain profile by ID
// @Tags         Toolchain Profiles
// @Produce      json
// @Param        id   path      string  true  "Toolchain Profile ID"
// @Success      200  {object}  db.ToolchainProfile
// @Failure      400  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/toolchains/{id} [get]
func (h *Handler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Toolchain profile ID required")
		return
	}

	profile, err := h.repo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if profile == nil {
		common.NotFound(c, "Toolchain profile not found")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// HandleCreate creates a new user toolchain profile
// @Summary      Create a toolchain profile
// @Description  Creates a new user toolchain profile
// @Tags         Toolchain Profiles
// @Accept       json
// @Produce      json
// @Param        body  body      CreateToolchainProfileRequest  true  "Toolchain profile data"
// @Success      201   {object}  db.ToolchainProfile
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/toolchains [post]
func (h *Handler) HandleCreate(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	var req CreateToolchainProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	// Check for duplicate name
	existing, err := h.repo.GetByName(req.Name)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if existing != nil {
		common.BadRequest(c, "Toolchain profile name already exists: "+req.Name)
		return
	}

	profile := &db.ToolchainProfile{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		IsSystem:    false,
		OwnerID:     claims.UserID,
	}

	if err := h.repo.Create(profile); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// HandleUpdate updates an existing toolchain profile
// @Summary      Update a toolchain profile
// @Description  Updates an existing toolchain profile (owner or admin only)
// @Tags         Toolchain Profiles
// @Accept       json
// @Produce      json
// @Param        id    path      string                          true  "Toolchain Profile ID"
// @Param        body  body      UpdateToolchainProfileRequest   true  "Toolchain profile data"
// @Success      200   {object}  db.ToolchainProfile
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/toolchains/{id} [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Toolchain profile ID required")
		return
	}

	profile, err := h.repo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if profile == nil {
		common.NotFound(c, "Toolchain profile not found")
		return
	}

	// Permission check: system profiles require admin, user profiles require ownership
	if profile.IsSystem && !claims.HasAdminAccess() {
		common.Forbidden(c, "Admin access required to modify system profiles")
		return
	}
	if !profile.IsSystem && profile.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only modify your own profiles")
		return
	}

	var req UpdateToolchainProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name != profile.Name {
			existing, err := h.repo.GetByName(*req.Name)
			if err != nil {
				common.InternalError(c, err.Error())
				return
			}
			if existing != nil {
				common.BadRequest(c, "Toolchain profile name already exists: "+*req.Name)
				return
			}
		}
		profile.Name = *req.Name
	}
	if req.DisplayName != nil {
		profile.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.Config != nil {
		profile.Config = *req.Config
	}

	if err := h.repo.Update(profile); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, profile)
}

// HandleDelete deletes a toolchain profile
// @Summary      Delete a toolchain profile
// @Description  Deletes a toolchain profile (owner or admin only, system profiles cannot be deleted)
// @Tags         Toolchain Profiles
// @Produce      json
// @Param        id   path      string  true  "Toolchain Profile ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/toolchains/{id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Toolchain profile ID required")
		return
	}

	profile, err := h.repo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if profile == nil {
		common.NotFound(c, "Toolchain profile not found")
		return
	}

	// System profiles cannot be deleted
	if profile.IsSystem {
		common.Forbidden(c, "System toolchain profiles cannot be deleted")
		return
	}

	// Permission check: user profiles require ownership or admin
	if profile.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only delete your own profiles")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
