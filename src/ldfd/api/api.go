package api

import (
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

var log *logs.Logger

// VersionInfo holds the server version information
var VersionInfo *version.Info

// SetLogger sets the logger for the api package
func SetLogger(l *logs.Logger) {
	log = l
}

// SetVersionInfo sets the version info for the api package
func SetVersionInfo(v *version.Info) {
	VersionInfo = v
}

// API holds all dependencies needed by HTTP handlers
type API struct {
	// Repositories
	distRepo          *db.DistributionRepository
	sourceRepo        *db.SourceRepository
	componentRepo     *db.ComponentRepository
	sourceVersionRepo *db.SourceVersionRepository
	langPackRepo      *db.LanguagePackRepository
	database          *db.Database

	// Services
	storage          storage.Backend
	authHandler      *auth.Handler
	jwtService       *auth.JWTService
	downloadManager  *download.Manager
	versionDiscovery *download.VersionDiscovery
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
	AuthHandler       *auth.Handler
	JWTService        *auth.JWTService
	DownloadManager   *download.Manager
	VersionDiscovery  *download.VersionDiscovery
}

// New creates a new API instance
func New(cfg Config) *API {
	return &API{
		distRepo:          cfg.DistRepo,
		sourceRepo:        cfg.SourceRepo,
		componentRepo:     cfg.ComponentRepo,
		sourceVersionRepo: cfg.SourceVersionRepo,
		langPackRepo:      cfg.LangPackRepo,
		database:          cfg.Database,
		storage:           cfg.Storage,
		authHandler:       cfg.AuthHandler,
		jwtService:        cfg.JWTService,
		downloadManager:   cfg.DownloadManager,
		versionDiscovery:  cfg.VersionDiscovery,
	}
}
