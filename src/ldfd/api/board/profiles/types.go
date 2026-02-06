package profiles

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles board profile HTTP requests
type Handler struct {
	boardProfileRepo *db.BoardProfileRepository
}

// Config contains configuration options for the Handler
type Config struct {
	BoardProfileRepo *db.BoardProfileRepository
}

// CreateBoardProfileRequest represents the request to create a board profile
type CreateBoardProfileRequest struct {
	Name        string         `json:"name" binding:"required" example:"jetson-orin"`
	DisplayName string         `json:"display_name" binding:"required" example:"NVIDIA Jetson Orin"`
	Description string         `json:"description" example:"NVIDIA Jetson Orin developer kit"`
	Arch        string         `json:"arch" binding:"required" example:"aarch64"`
	Config      db.BoardConfig `json:"config"`
}

// UpdateBoardProfileRequest represents the request to update a board profile
type UpdateBoardProfileRequest struct {
	Name        *string         `json:"name" example:"jetson-orin"`
	DisplayName *string         `json:"display_name" example:"NVIDIA Jetson Orin"`
	Description *string         `json:"description" example:"NVIDIA Jetson Orin developer kit"`
	Config      *db.BoardConfig `json:"config"`
}

// BoardProfileListResponse represents a list of board profiles
type BoardProfileListResponse struct {
	Count    int               `json:"count" example:"2"`
	Profiles []db.BoardProfile `json:"profiles"`
}
