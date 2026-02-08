package toolchains

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles toolchain profile HTTP requests
type Handler struct {
	repo *db.ToolchainProfileRepository
}

// Config contains configuration options for the Handler
type Config struct {
	ToolchainProfileRepo *db.ToolchainProfileRepository
}

// CreateToolchainProfileRequest represents the request to create a toolchain profile
type CreateToolchainProfileRequest struct {
	Name        string             `json:"name" binding:"required" example:"gcc-cross-aarch64"`
	DisplayName string             `json:"display_name" binding:"required" example:"GCC Cross (aarch64)"`
	Description string             `json:"description" example:"GCC cross-compiler for aarch64 targets"`
	Type        string             `json:"type" binding:"required" example:"gcc"`
	Config      db.ToolchainConfig `json:"config"`
}

// UpdateToolchainProfileRequest represents the request to update a toolchain profile
type UpdateToolchainProfileRequest struct {
	Name        *string             `json:"name" example:"gcc-cross-aarch64"`
	DisplayName *string             `json:"display_name" example:"GCC Cross (aarch64)"`
	Description *string             `json:"description" example:"GCC cross-compiler for aarch64 targets"`
	Config      *db.ToolchainConfig `json:"config"`
}

// ToolchainProfileListResponse represents a list of toolchain profiles
type ToolchainProfileListResponse struct {
	Count    int                   `json:"count" example:"2"`
	Profiles []db.ToolchainProfile `json:"toolchain_profiles"`
}
