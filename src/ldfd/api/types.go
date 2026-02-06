package api

import (
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
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/forge"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// ErrorResponse is an alias to common.ErrorResponse for backwards compatibility
type ErrorResponse = common.ErrorResponse

// API holds all handler instances and dependencies
type API struct {
	// Subpackage handlers
	Base          *base.Handler
	Auth          *apiauth.Handler
	Distributions *distributions.Handler
	Components    *components.Handler
	Sources       *sources.Handler
	Downloads     *downloads.Handler
	Mirrors       *downloads.MirrorHandler
	Builds        *builds.Handler
	Artifacts     *artifacts.Handler
	Branding      *branding.Handler
	LangPacks     *langpacks.Handler
	Settings      *settings.Handler
	Forge         *apiforge.Handler
	BoardProfiles *boardprofiles.Handler

	// Direct dependencies for middleware
	jwtService    *auth.JWTService
	storage       storage.Backend
	forgeRegistry *forge.Registry
}

// Config contains API configuration options
type Config struct {
	DistRepo          *db.DistributionRepository
	SourceRepo        *db.SourceRepository
	ComponentRepo     *db.ComponentRepository
	SourceVersionRepo *db.SourceVersionRepository
	DownloadJobRepo   *db.DownloadJobRepository
	MirrorConfigRepo  *db.MirrorConfigRepository
	LangPackRepo      *db.LanguagePackRepository
	BoardProfileRepo  *db.BoardProfileRepository
	Database          *db.Database
	Storage           storage.Backend
	UserManager       *auth.UserManager
	JWTService        *auth.JWTService
	DownloadManager   *download.Manager
	BuildManager      *build.Manager
	VersionDiscovery  *download.VersionDiscovery
	ForgeRegistry     *forge.Registry
}
