package auth

import (
	"net/http"

	"github.com/bitswalk/ldf/src/common/errors"
	coreauth "github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

// HandleListRoles returns all available roles
// @Summary      List all roles
// @Description  Returns all available user roles.
// @Tags         Roles
// @Produce      json
// @Success      200  {object}  object  "List of roles"
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/roles [get]
func (h *Handler) HandleListRoles(c *gin.Context) {
	roles, err := h.userManager.ListRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
	})
}

// HandleGetRole returns a specific role by ID
// @Summary      Get a role by ID
// @Description  Returns a specific role identified by its ID.
// @Tags         Roles
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  object  "Role details"
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/roles/{id} [get]
func (h *Handler) HandleGetRole(c *gin.Context) {
	id := c.Param("id")

	role, err := h.userManager.GetRoleByID(id)
	if err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// HandleCreateRole creates a new custom role
// @Summary      Create a new role
// @Description  Creates a new custom role with the specified permissions.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        request  body      RoleCreateRequest  true  "Role creation request"
// @Success      201      {object}  object             "Created role"
// @Failure      400      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/roles [post]
func (h *Handler) HandleCreateRole(c *gin.Context) {
	var req RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	if req.ParentRoleID != "" {
		_, err := h.userManager.GetRoleByID(req.ParentRoleID)
		if err != nil {
			if errors.Is(err, errors.ErrRoleNotFound) {
				c.JSON(http.StatusBadRequest, errors.ErrRoleNotFound.WithMessage("Parent role not found").ToResponse())
				return
			}
			c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
			return
		}
	}

	role := coreauth.NewRole(req.Name, req.Description, coreauth.RolePermissions{
		CanRead:   req.Permissions.CanRead,
		CanWrite:  req.Permissions.CanWrite,
		CanDelete: req.Permissions.CanDelete,
		CanAdmin:  req.Permissions.CanAdmin,
	}, req.ParentRoleID)

	if err := h.userManager.CreateRole(role); err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"role": role,
	})
}

// HandleUpdateRole updates an existing custom role
// @Summary      Update a role
// @Description  Updates an existing custom role identified by its ID.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        id       path      string             true  "Role ID"
// @Param        request  body      RoleUpdateRequest  true  "Role update request"
// @Success      200      {object}  object             "Updated role"
// @Failure      400      {object}  common.ErrorResponse
// @Failure      404      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/roles/{id} [put]
func (h *Handler) HandleUpdateRole(c *gin.Context) {
	id := c.Param("id")

	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	role, err := h.userManager.GetRoleByID(id)
	if err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Permissions != nil {
		role.Permissions = coreauth.RolePermissions{
			CanRead:   req.Permissions.CanRead,
			CanWrite:  req.Permissions.CanWrite,
			CanDelete: req.Permissions.CanDelete,
			CanAdmin:  req.Permissions.CanAdmin,
		}
	}

	if err := h.userManager.UpdateRole(role); err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// HandleDeleteRole deletes a custom role
// @Summary      Delete a role
// @Description  Deletes a custom role identified by its ID.
// @Tags         Roles
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  object  "Deletion confirmation"
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/roles/{id} [delete]
func (h *Handler) HandleDeleteRole(c *gin.Context) {
	id := c.Param("id")

	if err := h.userManager.DeleteRole(id); err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role deleted successfully",
	})
}
