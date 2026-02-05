package client

import "context"

// LoginResponse represents the login API response
type LoginResponse struct {
	User struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Role   string `json:"role"`
		RoleID string `json:"role_id"`
	} `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
}

// ValidateResponse represents the validate API response
type ValidateResponse struct {
	Valid bool `json:"valid"`
	User  struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Email       string          `json:"email"`
		Role        string          `json:"role"`
		RoleID      string          `json:"role_id"`
		Permissions map[string]bool `json:"permissions"`
	} `json:"user"`
}

// AuthRequest builds the nested auth request body used by the server
type AuthRequest struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name     string `json:"name"`
					Password string `json:"password"`
					Email    string `json:"email,omitempty"`
					Role     string `json:"role,omitempty"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
	} `json:"auth"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Login authenticates with the server and returns tokens
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	var req AuthRequest
	req.Auth.Identity.Methods = []string{"password"}
	req.Auth.Identity.Password.User.Name = username
	req.Auth.Identity.Password.User.Password = password

	var resp LoginResponse
	if err := c.Post(ctx, "/auth/login", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUser registers a new user account
func (c *Client) CreateUser(ctx context.Context, name, password, email, role string) (*LoginResponse, error) {
	var req AuthRequest
	req.Auth.Identity.Methods = []string{"password"}
	req.Auth.Identity.Password.User.Name = name
	req.Auth.Identity.Password.User.Password = password
	req.Auth.Identity.Password.User.Email = email
	req.Auth.Identity.Password.User.Role = role

	var resp LoginResponse
	if err := c.Post(ctx, "/auth/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout revokes the current token
func (c *Client) Logout(ctx context.Context) error {
	return c.Post(ctx, "/auth/logout", nil, nil)
}

// Refresh exchanges a refresh token for new tokens
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	req := RefreshRequest{RefreshToken: refreshToken}
	var resp LoginResponse
	if err := c.Post(ctx, "/auth/refresh", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Validate checks the current token and returns user info
func (c *Client) Validate(ctx context.Context) (*ValidateResponse, error) {
	var resp ValidateResponse
	if err := c.Get(ctx, "/auth/validate", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
