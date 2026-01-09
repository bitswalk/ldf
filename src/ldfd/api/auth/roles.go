package auth

import (
	"net/http"

	"github.com/bitswalk/ldf/src/common/errors"
	coreauth "github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

// HandleListRoles returns all available roles
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
