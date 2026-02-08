package profiles

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new board profiles handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		boardProfileRepo: cfg.BoardProfileRepo,
	}
}

// HandleList returns all board profiles, optionally filtered by architecture
// @Summary      List board profiles
// @Description  Returns all board profiles, optionally filtered by architecture
// @Tags         Board Profiles
// @Produce      json
// @Param        arch  query     string  false  "Filter by architecture (x86_64, aarch64)"
// @Success      200   {object}  BoardProfileListResponse
// @Failure      500   {object}  common.ErrorResponse
// @Router       /v1/board/profiles [get]
func (h *Handler) HandleList(c *gin.Context) {
	var profiles []db.BoardProfile
	var err error

	if arch := c.Query("arch"); arch != "" {
		profiles, err = h.boardProfileRepo.ListByArch(db.TargetArch(arch))
	} else {
		profiles, err = h.boardProfileRepo.List()
	}

	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if profiles == nil {
		profiles = []db.BoardProfile{}
	}

	c.JSON(http.StatusOK, BoardProfileListResponse{
		Count:    len(profiles),
		Profiles: profiles,
	})
}

// HandleGet returns a single board profile by ID
// @Summary      Get a board profile
// @Description  Returns a single board profile by ID
// @Tags         Board Profiles
// @Produce      json
// @Param        id   path      string  true  "Board Profile ID"
// @Success      200  {object}  db.BoardProfile
// @Failure      400  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/board/profiles/{id} [get]
func (h *Handler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Board profile ID required")
		return
	}

	profile, err := h.boardProfileRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if profile == nil {
		common.NotFound(c, "Board profile not found")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// HandleCreate creates a new user board profile
// @Summary      Create a board profile
// @Description  Creates a new user board profile
// @Tags         Board Profiles
// @Accept       json
// @Produce      json
// @Param        body  body      CreateBoardProfileRequest  true  "Board profile data"
// @Success      201   {object}  db.BoardProfile
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/board/profiles [post]
func (h *Handler) HandleCreate(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	var req CreateBoardProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	arch := db.TargetArch(req.Arch)

	// Check for duplicate name
	existing, err := h.boardProfileRepo.GetByName(req.Name)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if existing != nil {
		common.BadRequest(c, "Board profile name already exists: "+req.Name)
		return
	}

	profile := &db.BoardProfile{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Arch:        arch,
		Config:      req.Config,
		IsSystem:    false,
		OwnerID:     claims.UserID,
	}

	if err := h.boardProfileRepo.Create(profile); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// HandleUpdate updates an existing board profile
// @Summary      Update a board profile
// @Description  Updates an existing board profile (owner or admin only)
// @Tags         Board Profiles
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "Board Profile ID"
// @Param        body  body      UpdateBoardProfileRequest  true  "Board profile data"
// @Success      200   {object}  db.BoardProfile
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/board/profiles/{id} [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Board profile ID required")
		return
	}

	profile, err := h.boardProfileRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if profile == nil {
		common.NotFound(c, "Board profile not found")
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

	var req UpdateBoardProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	// Apply updates
	if req.Name != nil {
		// Check for duplicate name (if changing)
		if *req.Name != profile.Name {
			existing, err := h.boardProfileRepo.GetByName(*req.Name)
			if err != nil {
				common.InternalError(c, err.Error())
				return
			}
			if existing != nil {
				common.BadRequest(c, "Board profile name already exists: "+*req.Name)
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

	if err := h.boardProfileRepo.Update(profile); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, profile)
}

// HandleDelete deletes a board profile
// @Summary      Delete a board profile
// @Description  Deletes a board profile (owner or admin only, system profiles cannot be deleted)
// @Tags         Board Profiles
// @Produce      json
// @Param        id   path      string  true  "Board Profile ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/board/profiles/{id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Board profile ID required")
		return
	}

	profile, err := h.boardProfileRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if profile == nil {
		common.NotFound(c, "Board profile not found")
		return
	}

	// System profiles cannot be deleted
	if profile.IsSystem {
		common.Forbidden(c, "System board profiles cannot be deleted")
		return
	}

	// Permission check: user profiles require ownership or admin
	if profile.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only delete your own profiles")
		return
	}

	if err := h.boardProfileRepo.Delete(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
