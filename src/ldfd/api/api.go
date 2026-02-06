package api

import (
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfd/api/artifacts"
	apiauth "github.com/bitswalk/ldf/src/ldfd/api/auth"
	"github.com/bitswalk/ldf/src/ldfd/api/base"
	"github.com/bitswalk/ldf/src/ldfd/api/branding"
	"github.com/bitswalk/ldf/src/ldfd/api/builds"
	"github.com/bitswalk/ldf/src/ldfd/api/components"
	"github.com/bitswalk/ldf/src/ldfd/api/distributions"
	"github.com/bitswalk/ldf/src/ldfd/api/downloads"
	apiforge "github.com/bitswalk/ldf/src/ldfd/api/forge"
	"github.com/bitswalk/ldf/src/ldfd/api/langpacks"
	"github.com/bitswalk/ldf/src/ldfd/api/settings"
	"github.com/bitswalk/ldf/src/ldfd/api/sources"
	"github.com/bitswalk/ldf/src/ldfd/db"
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

// New creates a new API instance with all subpackage handlers
func New(cfg Config) *API {
	return &API{
		Base: base.NewHandler(),

		Auth: apiauth.NewHandler(apiauth.Config{
			UserManager: cfg.UserManager,
			JWTService:  cfg.JWTService,
		}),

		Distributions: distributions.NewHandler(distributions.Config{
			DistRepo:        cfg.DistRepo,
			DownloadJobRepo: cfg.DownloadJobRepo,
			SourceRepo:      cfg.SourceRepo,
			JWTService:      cfg.JWTService,
			StorageManager:  newStorageManager(cfg.Storage),
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

		Builds: builds.NewHandler(builds.Config{
			DistRepo:     cfg.DistRepo,
			BuildManager: cfg.BuildManager,
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

		Forge: apiforge.NewHandler(apiforge.Config{
			Registry: cfg.ForgeRegistry,
		}),

		jwtService:    cfg.JWTService,
		storage:       cfg.Storage,
		forgeRegistry: cfg.ForgeRegistry,
	}
}

// newStorageManager creates a StorageManager from a storage backend,
// returning a proper nil interface when the backend is nil.
func newStorageManager(backend storage.Backend) distributions.StorageManager {
	if backend == nil {
		return nil
	}
	return distributions.NewStorageAdapter(backend)
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
