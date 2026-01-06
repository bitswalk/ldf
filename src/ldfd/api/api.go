package api

import (
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfd/api/artifacts"
	apiauth "github.com/bitswalk/ldf/src/ldfd/api/auth"
	"github.com/bitswalk/ldf/src/ldfd/api/base"
	"github.com/bitswalk/ldf/src/ldfd/api/branding"
	"github.com/bitswalk/ldf/src/ldfd/api/components"
	"github.com/bitswalk/ldf/src/ldfd/api/distributions"
	"github.com/bitswalk/ldf/src/ldfd/api/downloads"
	"github.com/bitswalk/ldf/src/ldfd/api/langpacks"
	"github.com/bitswalk/ldf/src/ldfd/api/settings"
	"github.com/bitswalk/ldf/src/ldfd/api/sources"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// SetLogger sets the logger for the api package and subpackages
func SetLogger(l *logs.Logger) {
	distributions.SetLogger(l)
	sources.SetLogger(l)
	settings.SetLogger(l)
	apiauth.SetLogger(l)
}

// SetVersionInfo sets the version info for the api package and subpackages
func SetVersionInfo(v *version.Info) {
	base.SetVersionInfo(v)
}

// API holds all handler instances and dependencies
type API struct {
	// Subpackage handlers
	Base          *base.Handler
	Auth          *apiauth.Handler
	Distributions *distributions.Handler
	Components    *components.Handler
	Sources       *sources.Handler
	Downloads     *downloads.Handler
	Artifacts     *artifacts.Handler
	Branding      *branding.Handler
	LangPacks     *langpacks.Handler
	Settings      *settings.Handler

	// Direct dependencies for middleware
	jwtService *auth.JWTService
	storage    storage.Backend
}

// Config contains API configuration options
type Config struct {
	DistRepo          *db.DistributionRepository
	SourceRepo        *db.SourceRepository
	ComponentRepo     *db.ComponentRepository
	SourceVersionRepo *db.SourceVersionRepository
	LangPackRepo      *db.LanguagePackRepository
	Database          *db.Database
	Storage           storage.Backend
	UserManager       *auth.UserManager
	JWTService        *auth.JWTService
	DownloadManager   *download.Manager
	VersionDiscovery  *download.VersionDiscovery
}

// New creates a new API instance with all subpackage handlers
func New(cfg Config) *API {
	return &API{
		Base: base.NewHandler(),

		Auth: apiauth.NewHandler(apiauth.Config{
			UserManager: cfg.UserManager,
			JWTService:  cfg.JWTService,
		}),

		Distributions: distributions.NewHandler(distributions.Config{
			DistRepo:   cfg.DistRepo,
			JWTService: cfg.JWTService,
		}),

		Components: components.NewHandler(components.Config{
			ComponentRepo:     cfg.ComponentRepo,
			SourceVersionRepo: cfg.SourceVersionRepo,
		}),

		Sources: sources.NewHandler(sources.Config{
			SourceRepo:        cfg.SourceRepo,
			SourceVersionRepo: cfg.SourceVersionRepo,
			VersionDiscovery:  cfg.VersionDiscovery,
		}),

		Downloads: downloads.NewHandler(downloads.Config{
			DistRepo:        cfg.DistRepo,
			ComponentRepo:   cfg.ComponentRepo,
			DownloadManager: cfg.DownloadManager,
		}),

		Artifacts: artifacts.NewHandler(artifacts.Config{
			DistRepo:   cfg.DistRepo,
			Storage:    cfg.Storage,
			JWTService: cfg.JWTService,
		}),

		Branding: branding.NewHandler(branding.Config{
			Storage: cfg.Storage,
		}),

		LangPacks: langpacks.NewHandler(langpacks.Config{
			LangPackRepo: cfg.LangPackRepo,
		}),

		Settings: settings.NewHandler(settings.Config{
			Database: cfg.Database,
		}),

		jwtService: cfg.JWTService,
		storage:    cfg.Storage,
	}
}

// HasStorage returns true if storage backend is configured
func (a *API) HasStorage() bool {
	return a.storage != nil
}

// LoadConfigFromDatabase re-exports settings.LoadConfigFromDatabase for use by core/server.go
func LoadConfigFromDatabase(database *db.Database) error {
	return settings.LoadConfigFromDatabase(database)
}

// SyncConfigToDatabase re-exports settings.SyncConfigToDatabase for use by core/server.go
func SyncConfigToDatabase(database *db.Database) error {
	return settings.SyncConfigToDatabase(database)
}
