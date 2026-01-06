package api

import "github.com/gin-gonic/gin"

// RegisterRoutes configures all API routes on the given router
func (a *API) RegisterRoutes(router *gin.Engine) {
	// Root endpoint - API discovery
	router.GET("/", a.handleRoot)

	// Auth routes - delegate to auth.Handler
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/create", a.authHandler.HandleCreate)
		authGroup.POST("/login", a.authHandler.HandleLogin)
		authGroup.POST("/logout", a.authHandler.HandleLogout)
		authGroup.POST("/refresh", a.authHandler.HandleRefresh)
		authGroup.GET("/validate", a.authHandler.HandleValidate)
	}

	// Role routes - read operations (public, anyone can list roles)
	roles := router.Group("/v1/roles")
	{
		roles.GET("", a.authHandler.HandleListRoles)
		roles.GET("/:id", a.authHandler.HandleGetRole)
	}

	// Role routes - write operations (requires admin access)
	rolesAdmin := router.Group("/v1/roles")
	rolesAdmin.Use(a.adminAccessRequired())
	{
		rolesAdmin.POST("", a.authHandler.HandleCreateRole)
		rolesAdmin.PUT("/:id", a.authHandler.HandleUpdateRole)
		rolesAdmin.DELETE("/:id", a.authHandler.HandleDeleteRole)
	}

	// API v1 routes
	v1 := router.Group("/v1")
	{
		v1.GET("/health", a.handleHealth)
		v1.GET("/version", a.handleVersion)

		// Distribution routes - read operations (public)
		distributions := v1.Group("/distributions")
		{
			distributions.GET("", a.handleListDistributions)
			distributions.GET("/:id", a.handleGetDistribution)
			distributions.GET("/:id/logs", a.handleGetDistributionLogs)
			distributions.GET("/stats", a.handleGetDistributionStats)

			// Artifact read operations (public)
			if a.storage != nil {
				distributions.GET("/:id/artifacts", a.handleListArtifacts)
				distributions.GET("/:id/artifacts/*path", a.handleDownloadArtifact)
				distributions.GET("/:id/artifacts-url/*path", a.handleGetArtifactURL)
			}
		}

		// Distribution routes - write operations (requires auth with write access)
		distributionsWrite := v1.Group("/distributions")
		distributionsWrite.Use(a.writeAccessRequired())
		{
			distributionsWrite.POST("", a.handleCreateDistribution)
			distributionsWrite.PUT("/:id", a.handleUpdateDistribution)
			distributionsWrite.DELETE("/:id", a.handleDeleteDistribution)

			// Download operations (write access required)
			distributionsWrite.POST("/:id/downloads", a.handleStartDistributionDownloads)
			distributionsWrite.DELETE("/:id/downloads", a.handleFlushDistributionDownloads)

			// Artifact write operations (requires auth with write access)
			if a.storage != nil {
				distributionsWrite.POST("/:id/artifacts", a.handleUploadArtifact)
				distributionsWrite.DELETE("/:id/artifacts/*path", a.handleDeleteArtifact)
			}
		}

		// Storage and artifacts endpoints
		if a.storage != nil {
			v1.GET("/storage/status", a.handleStorageStatus)
			v1.GET("/artifacts", a.handleListAllArtifacts)
		}

		// Settings routes - root access only
		settings := v1.Group("/settings")
		settings.Use(a.rootAccessRequired())
		{
			settings.GET("", a.handleGetSettings)
			settings.GET("/*key", a.handleGetSetting)
			settings.PUT("/*key", a.handleUpdateSetting)
			settings.POST("/database/reset", a.handleResetDatabase)
		}

		// Language packs routes - read operations (authenticated)
		langPacks := v1.Group("/language-packs")
		langPacks.Use(a.authRequired())
		{
			langPacks.GET("", a.handleListLanguagePacks)
			langPacks.GET("/:locale", a.handleGetLanguagePack)
		}

		// Language packs routes - write operations (root only)
		langPacksAdmin := v1.Group("/language-packs")
		langPacksAdmin.Use(a.rootAccessRequired())
		{
			langPacksAdmin.POST("", a.handleUploadLanguagePack)
			langPacksAdmin.DELETE("/:locale", a.handleDeleteLanguagePack)
		}

		// Branding routes - read operations (public)
		if a.storage != nil {
			branding := v1.Group("/branding")
			{
				branding.GET("/:asset", a.handleGetBrandingAsset)
				branding.GET("/:asset/info", a.handleGetBrandingAssetInfo)
			}

			// Branding routes - write operations (root only)
			brandingAdmin := v1.Group("/branding")
			brandingAdmin.Use(a.rootAccessRequired())
			{
				brandingAdmin.POST("/:asset", a.handleUploadBrandingAsset)
				brandingAdmin.DELETE("/:asset", a.handleDeleteBrandingAsset)
			}
		}

		// Sources routes - authenticated users can read/manage their own sources
		sources := v1.Group("/sources")
		sources.Use(a.authRequired())
		{
			sources.GET("", a.handleListSources)
			sources.POST("", a.handleCreateUserSource)
			sources.PUT("/:id", a.handleUpdateUserSource)
			sources.DELETE("/:id", a.handleDeleteUserSource)
			sources.GET("/component/:componentId", a.handleListSourcesByComponent)

			// User source details and versions
			sources.GET("/user/:id", a.handleGetUserSourceByID)
			sources.GET("/user/:id/versions", a.handleListUserSourceVersions)
			sources.POST("/user/:id/sync", a.handleSyncUserSourceVersions)
			sources.GET("/user/:id/sync/status", a.handleGetUserSourceSyncStatus)

			// Default sources management - root only
			defaults := sources.Group("/defaults")
			defaults.Use(a.rootAccessRequired())
			{
				defaults.GET("", a.handleListDefaultSources)
				defaults.POST("", a.handleCreateDefaultSource)
				defaults.PUT("/:id", a.handleUpdateDefaultSource)
				defaults.DELETE("/:id", a.handleDeleteDefaultSource)
			}

			// Default source details (auth required for reading)
			sources.GET("/defaults/:id", a.handleGetDefaultSourceByID)
			sources.GET("/defaults/:id/versions", a.handleListDefaultSourceVersions)
			sources.GET("/defaults/:id/sync/status", a.handleGetDefaultSourceSyncStatus)
		}

		// Default source sync - root only (separate group for write operations)
		sourcesDefaultSync := v1.Group("/sources/defaults")
		sourcesDefaultSync.Use(a.rootAccessRequired())
		{
			sourcesDefaultSync.POST("/:id/sync", a.handleSyncDefaultSourceVersions)
		}

		// Component routes - read operations (public)
		components := v1.Group("/components")
		{
			components.GET("", a.handleListComponents)
			components.GET("/categories", a.handleGetComponentCategories)
			components.GET("/:id", a.handleGetComponent)
			components.GET("/:id/versions", a.handleGetComponentVersions)
			components.GET("/:id/resolve-version", a.handleResolveComponentVersion)
			components.GET("/category/:category", a.handleListComponentsByCategory)
		}

		// Component routes - write operations (root only)
		componentsAdmin := v1.Group("/components")
		componentsAdmin.Use(a.rootAccessRequired())
		{
			componentsAdmin.POST("", a.handleCreateComponent)
			componentsAdmin.PUT("/:id", a.handleUpdateComponent)
			componentsAdmin.DELETE("/:id", a.handleDeleteComponent)
		}

		// Distribution downloads - read (auth required)
		distDownloadsRead := v1.Group("/distributions/:id/downloads")
		distDownloadsRead.Use(a.authRequired())
		{
			distDownloadsRead.GET("", a.handleListDistributionDownloads)
		}

		// Download job routes - read (auth required)
		downloadsRead := v1.Group("/downloads")
		downloadsRead.Use(a.authRequired())
		{
			downloadsRead.GET("/:jobId", a.handleGetDownloadJob)
		}

		// Download job routes - write (write access required)
		downloadsWrite := v1.Group("/downloads")
		downloadsWrite.Use(a.writeAccessRequired())
		{
			downloadsWrite.POST("/:jobId/cancel", a.handleCancelDownload)
			downloadsWrite.POST("/:jobId/retry", a.handleRetryDownload)
		}

		// Active downloads - admin only
		downloadsAdmin := v1.Group("/downloads")
		downloadsAdmin.Use(a.adminAccessRequired())
		{
			downloadsAdmin.GET("/active", a.handleListActiveDownloads)
		}
	}
}
