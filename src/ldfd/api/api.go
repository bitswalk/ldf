package api

import (
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfd/api/artifacts"
	apiauth "github.com/bitswalk/ldf/src/ldfd/api/auth"
	"github.com/bitswalk/ldf/src/ldfd/api/base"
	boardprofiles "github.com/bitswalk/ldf/src/ldfd/api/board/profiles"
	"github.com/bitswalk/ldf/src/ldfd/api/branding"
	"github.com/bitswalk/ldf/src/ldfd/api/builds"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/api/components"
	"github.com/bitswalk/ldf/src/ldfd/api/distributions"
	"github.com/bitswalk/ldf/src/ldfd/api/downloads"
	apiforge "github.com/bitswalk/ldf/src/ldfd/api/forge"
	"github.com/bitswalk/ldf/src/ldfd/api/langpacks"
	"github.com/bitswalk/ldf/src/ldfd/api/settings"
	"github.com/bitswalk/ldf/src/ldfd/api/sources"
	"github.com/bitswalk/ldf/src/ldfd/api/toolchains"
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/security"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// SetLogger sets the logger for the api package and subpackages
func SetLogger(l *logs.Logger) {
	distributions.SetLogger(l)
	sources.SetLogger(l)
	settings.SetLogger(l)
	apiauth.SetLogger(l)
	common.SetAuditLogger(l)
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
			BuildJobRepo:    buildJobRepoFromManager(cfg.BuildManager),
			SourceRepo:      cfg.SourceRepo,
			JWTService:      cfg.JWTService,
			StorageManager:  newStorageManager(cfg.Storage),
			KernelConfigSvc: build.NewKernelConfigService(cfg.Storage),
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

		Mirrors: downloads.NewMirrorHandler(cfg.MirrorConfigRepo),

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
			Database:      cfg.Database,
			SecretManager: cfg.SecretManager,
		}),

		Forge: apiforge.NewHandler(apiforge.Config{
			Registry: cfg.ForgeRegistry,
		}),

		BoardProfiles: boardprofiles.NewHandler(boardprofiles.Config{
			BoardProfileRepo: cfg.BoardProfileRepo,
		}),

		ToolchainProfiles: toolchains.NewHandler(toolchains.Config{
			ToolchainProfileRepo: cfg.ToolchainProfileRepo,
		}),

		jwtService:    cfg.JWTService,
		rateLimiter:   NewRateLimiter(cfg.RateLimitConfig),
		storage:       cfg.Storage,
		forgeRegistry: cfg.ForgeRegistry,
	}
}

// buildJobRepoFromManager safely extracts the BuildJobRepository from a build Manager,
// returning nil when the manager is nil (e.g. in tests).
func buildJobRepoFromManager(m *build.Manager) *db.BuildJobRepository {
	if m == nil {
		return nil
	}
	return m.BuildJobRepo()
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

// LoadConfigFromDatabase re-exports settings.LoadConfigFromDatabase for use by core/server.go.
// An optional SecretManager can be provided to decrypt sensitive settings.
func LoadConfigFromDatabase(database *db.Database, secrets ...interface{}) error {
	if len(secrets) > 0 {
		if sm, ok := secrets[0].(*security.SecretManager); ok {
			return settings.LoadConfigFromDatabase(database, sm)
		}
	}
	return settings.LoadConfigFromDatabase(database)
}

// SyncConfigToDatabase re-exports settings.SyncConfigToDatabase for use by core/server.go
func SyncConfigToDatabase(database *db.Database) error {
	return settings.SyncConfigToDatabase(database)
}
