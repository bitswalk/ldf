package base

// Handler handles base HTTP requests (root, health, version)
type Handler struct{}

// APIInfo represents the root API discovery response
type APIInfo struct {
	Name        string           `json:"name" example:"ldfd"`
	Description string           `json:"description" example:"LDF Platform API Server"`
	Version     string           `json:"version" example:"1.0.0"`
	APIVersions []string         `json:"api_versions" example:"v1"`
	Endpoints   APIInfoEndpoints `json:"endpoints"`
}

// APIInfoEndpoints contains the available API endpoints
type APIInfoEndpoints struct {
	Health  string        `json:"health" example:"/v1/health"`
	Version string        `json:"version" example:"/v1/version"`
	APIv1   string        `json:"api_v1" example:"/v1/"`
	Auth    AuthEndpoints `json:"auth"`
}

// AuthEndpoints contains the authentication endpoints
type AuthEndpoints struct {
	Create   string `json:"create" example:"/auth/create"`
	Login    string `json:"login" example:"/auth/login"`
	Logout   string `json:"logout" example:"/auth/logout"`
	Refresh  string `json:"refresh" example:"/auth/refresh"`
	Validate string `json:"validate" example:"/auth/validate"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"healthy"`
	Timestamp string `json:"timestamp" example:"2024-01-15T10:30:00Z"`
}

// VersionResponse represents the version information response
type VersionResponse struct {
	Version        string `json:"version" example:"Phoenix (2025.12) - v1.0.0-4f9f297"`
	ReleaseName    string `json:"release_name" example:"Phoenix"`
	ReleaseVersion string `json:"release_version" example:"1.0.0"`
	BuildDate      string `json:"build_date" example:"2024-01-15T10:30:00Z"`
	GitCommit      string `json:"git_commit" example:"4f9f297"`
	GoVersion      string `json:"go_version" example:"go1.24"`
}
