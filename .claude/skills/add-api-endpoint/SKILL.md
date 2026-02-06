---
name: add-api-endpoint
description: Scaffold a new API endpoint for ldfd with handler, types, route registration, and Swagger annotations. Use when adding a new API resource or endpoint to the server.
argument-hint: "[domain-name] [description]"

---

# Add API Endpoint

Scaffold a complete API endpoint following the project's established patterns.

## Arguments

- `$ARGUMENTS[0]` -- Domain name (e.g., `boards`, `profiles`). This becomes the package name and route group.
- `$ARGUMENTS[1]` -- Short description (e.g., "Board profile management")

## Steps

### 1. Create the handler package

Create `src/ldfd/api/$0/` with two files:

**`types.go`** -- Handler struct, Config struct, request/response types:

```go
package $0

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles $0-related HTTP requests
type Handler struct {
	// Add repository fields as needed, e.g.:
	// repo *db.XxxRepository
}

// Config contains configuration options for the Handler
type Config struct {
	// Match Handler fields, e.g.:
	// Repo *db.XxxRepository
}

// XxxListResponse represents a list response
type XxxListResponse struct {
	Count int       `json:"count" example:"10"`
	Items []db.Xxx  `json:"items"`
}

// CreateXxxRequest represents the creation request
type CreateXxxRequest struct {
	Name string `json:"name" binding:"required" example:"example"`
	// Add fields with json tags, binding tags for required, example tags for Swagger
}

// UpdateXxxRequest represents the update request (no binding:"required")
type UpdateXxxRequest struct {
	Name string `json:"name" example:"example"`
}
```

**`$0.go`** -- Constructor and handler methods with Swagger annotations:

```go
package $0

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		// Wire config fields to handler fields
	}
}

// HandleList returns all items
// @Summary      List items
// @Description  Returns all items
// @Tags         TagName
// @Produce      json
// @Success      200  {object}  XxxListResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/$0 [get]
func (h *Handler) HandleList(c *gin.Context) {
	// Implementation
}

// HandleGet returns a single item by ID
// @Summary      Get an item
// @Description  Returns a single item by ID
// @Tags         TagName
// @Produce      json
// @Param        id   path      string  true  "Item ID"
// @Success      200  {object}  db.Xxx
// @Failure      400  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/$0/{id} [get]
func (h *Handler) HandleGet(c *gin.Context) {
	// Implementation
}

// HandleCreate creates a new item
// @Summary      Create an item
// @Description  Creates a new item
// @Tags         TagName
// @Accept       json
// @Produce      json
// @Param        request  body      CreateXxxRequest  true  "Creation request"
// @Success      201      {object}  db.Xxx
// @Failure      400      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/$0 [post]
func (h *Handler) HandleCreate(c *gin.Context) {
	// Implementation
}
```

### 2. Register routes

Edit `src/ldfd/api/routes.go` and add the new route group inside the `v1` group:

```go
// Read operations (public)
xxxGroup := v1.Group("/$0")
{
    xxxGroup.GET("", a.Xxx.HandleList)
    xxxGroup.GET("/:id", a.Xxx.HandleGet)
}

// Write operations (protected)
xxxAdmin := v1.Group("/$0")
xxxAdmin.Use(a.rootAccessRequired())
{
    xxxAdmin.POST("", a.Xxx.HandleCreate)
    xxxAdmin.PUT("/:id", a.Xxx.HandleUpdate)
    xxxAdmin.DELETE("/:id", a.Xxx.HandleDelete)
}
```

### 3. Wire handler in API struct

Find the API struct (in `src/ldfd/api/` or `src/ldfd/core/server.go`) and add the new Handler field. Initialize it in the constructor using `NewHandler(Config{...})`.

### 4. Regenerate Swagger

Run: `~/go/bin/swag init --dir src/ldfd,src/common -g docs.go -o src/ldfd/docs --parseDependency --parseInternal`

### 5. Verify

Run: `task build:srv` to confirm compilation.

## Conventions to follow

- Error responses always use `common.ErrorResponse{Error: "...", Code: statusCode, Message: "..."}`
- Nil slices must be initialized: `if items == nil { items = []Type{} }`
- Path params extracted with `c.Param("id")`
- Query params with `c.Query("name")`
- Request body binding with `c.ShouldBindJSON(&req)`
- Status codes: 200 OK, 201 Created, 204 No Content, 400 Bad Request, 404 Not Found, 409 Conflict, 500 Internal Error
- Swagger `@Tags` should be PascalCase plural (e.g., "Boards", "Components")
- All write endpoints need `@Security BearerAuth`
- Pagination uses `common.GetPaginationParams(c, defaultLimit)`
