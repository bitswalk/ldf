package api

import "github.com/gin-gonic/gin"

// RegisterRoutes configures all API routes on the given router
func (a *API) RegisterRoutes(router *gin.Engine) {
	// Root endpoint - API discovery
	router.GET("/", a.Base.HandleRoot)

	// Auth routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/create", a.Auth.HandleCreate)
		authGroup.POST("/login", a.Auth.HandleLogin)
		authGroup.POST("/logout", a.Auth.HandleLogout)
		authGroup.POST("/refresh", a.Auth.HandleRefresh)
		authGroup.GET("/validate", a.Auth.HandleValidate)
	}

	// Role routes - read operations (public, anyone can list roles)
	roles := router.Group("/v1/roles")
	{
		roles.GET("", a.Auth.HandleListRoles)
		roles.GET("/:id", a.Auth.HandleGetRole)
	}

	// Role routes - write operations (requires admin access)
	rolesAdmin := router.Group("/v1/roles")
	rolesAdmin.Use(a.adminAccessRequired())
	{
		rolesAdmin.POST("", a.Auth.HandleCreateRole)
		rolesAdmin.PUT("/:id", a.Auth.HandleUpdateRole)
		rolesAdmin.DELETE("/:id", a.Auth.HandleDeleteRole)
	}

	// API v1 routes
	v1 := router.Group("/v1")
	{
		v1.GET("/health", a.Base.HandleHealth)
		v1.GET("/version", a.Base.HandleVersion)

		// Distribution routes - read operations (public)
		distributions := v1.Group("/distributions")
		{
			distributions.GET("", a.Distributions.HandleList)
			distributions.GET("/:id", a.Distributions.HandleGet)
			distributions.GET("/:id/logs", a.Distributions.HandleGetLogs)
			distributions.GET("/stats", a.Distributions.HandleGetStats)

			// Artifact read operations (public)
			if a.HasStorage() {
				distributions.GET("/:id/artifacts", a.Artifacts.HandleList)
				distributions.GET("/:id/artifacts/*path", a.Artifacts.HandleDownload)
				distributions.GET("/:id/artifacts-url/*path", a.Artifacts.HandleGetURL)
			}
		}

		// Distribution routes - write operations (requires auth with write access)
		distributionsWrite := v1.Group("/distributions")
		distributionsWrite.Use(a.writeAccessRequired())
		{
			distributionsWrite.POST("", a.Distributions.HandleCreate)
			distributionsWrite.PUT("/:id", a.Distributions.HandleUpdate)
			distributionsWrite.DELETE("/:id", a.Distributions.HandleDelete)

			// Download operations (write access required)
			distributionsWrite.POST("/:id/downloads", a.Downloads.HandleStartDistributionDownloads)
			distributionsWrite.DELETE("/:id/downloads", a.Downloads.HandleFlushDistributionDownloads)

			// Artifact write operations (requires auth with write access)
			if a.HasStorage() {
				distributionsWrite.POST("/:id/artifacts", a.Artifacts.HandleUpload)
				distributionsWrite.DELETE("/:id/artifacts/*path", a.Artifacts.HandleDelete)
			}
		}

		// Storage and artifacts endpoints
		if a.HasStorage() {
			v1.GET("/storage/status", a.Artifacts.HandleStorageStatus)
			v1.GET("/artifacts", a.Artifacts.HandleListAll)
		}

		// Settings routes - root access only
		settingsGroup := v1.Group("/settings")
		settingsGroup.Use(a.rootAccessRequired())
		{
			settingsGroup.GET("", a.Settings.HandleGetAll)
			settingsGroup.GET("/*key", a.Settings.HandleGet)
			settingsGroup.PUT("/*key", a.Settings.HandleUpdate)
			settingsGroup.POST("/database/reset", a.Settings.HandleResetDatabase)
		}

		// Language packs routes - read operations (authenticated)
		langPacks := v1.Group("/language-packs")
		langPacks.Use(a.authRequired())
		{
			langPacks.GET("", a.LangPacks.HandleList)
			langPacks.GET("/:locale", a.LangPacks.HandleGet)
		}

		// Language packs routes - write operations (root only)
		langPacksAdmin := v1.Group("/language-packs")
		langPacksAdmin.Use(a.rootAccessRequired())
		{
			langPacksAdmin.POST("", a.LangPacks.HandleUpload)
			langPacksAdmin.DELETE("/:locale", a.LangPacks.HandleDelete)
		}

		// Branding routes - read operations (public)
		if a.HasStorage() {
			brandingGroup := v1.Group("/branding")
			{
				brandingGroup.GET("/:asset", a.Branding.HandleGetAsset)
				brandingGroup.GET("/:asset/info", a.Branding.HandleGetAssetInfo)
			}

			// Branding routes - write operations (root only)
			brandingAdmin := v1.Group("/branding")
			brandingAdmin.Use(a.rootAccessRequired())
			{
				brandingAdmin.POST("/:asset", a.Branding.HandleUploadAsset)
				brandingAdmin.DELETE("/:asset", a.Branding.HandleDeleteAsset)
			}
		}

		// Sources routes - authenticated users can read/manage their own sources
		sourcesGroup := v1.Group("/sources")
		sourcesGroup.Use(a.authRequired())
		{
			sourcesGroup.GET("", a.Sources.HandleList)
			sourcesGroup.POST("", a.Sources.HandleCreateUserSource)
			sourcesGroup.PUT("/:id", a.Sources.HandleUpdateUserSource)
			sourcesGroup.DELETE("/:id", a.Sources.HandleDeleteUserSource)
			sourcesGroup.GET("/component/:componentId", a.Sources.HandleListByComponent)

			// User source details and versions
			sourcesGroup.GET("/user/:id", a.Sources.HandleGetUserSourceByID)
			sourcesGroup.GET("/user/:id/versions", a.Sources.HandleListUserVersions)
			sourcesGroup.POST("/user/:id/sync", a.Sources.HandleSyncUserVersions)
			sourcesGroup.GET("/user/:id/sync/status", a.Sources.HandleGetUserSyncStatus)

			// Default sources management - root only
			defaults := sourcesGroup.Group("/defaults")
			defaults.Use(a.rootAccessRequired())
			{
				defaults.GET("", a.Sources.HandleListDefaults)
				defaults.POST("", a.Sources.HandleCreateDefault)
				defaults.PUT("/:id", a.Sources.HandleUpdateDefault)
				defaults.DELETE("/:id", a.Sources.HandleDeleteDefault)
			}

			// Default source details (auth required for reading)
			sourcesGroup.GET("/defaults/:id", a.Sources.HandleGetDefaultByID)
			sourcesGroup.GET("/defaults/:id/versions", a.Sources.HandleListDefaultVersions)
			sourcesGroup.GET("/defaults/:id/sync/status", a.Sources.HandleGetDefaultSyncStatus)
		}

		// Default source sync - root only (separate group for write operations)
		sourcesDefaultSync := v1.Group("/sources/defaults")
		sourcesDefaultSync.Use(a.rootAccessRequired())
		{
			sourcesDefaultSync.POST("/:id/sync", a.Sources.HandleSyncDefaultVersions)
		}

		// Component routes - read operations (public)
		componentsGroup := v1.Group("/components")
		{
			componentsGroup.GET("", a.Components.HandleList)
			componentsGroup.GET("/categories", a.Components.HandleGetCategories)
			componentsGroup.GET("/kernel-modules", a.Components.HandleListKernelModules)
			componentsGroup.GET("/userspace", a.Components.HandleListUserspace)
			componentsGroup.GET("/hybrid", a.Components.HandleListHybrid)
			componentsGroup.GET("/:id", a.Components.HandleGet)
			componentsGroup.GET("/:id/versions", a.Components.HandleGetVersions)
			componentsGroup.GET("/:id/resolve-version", a.Components.HandleResolveVersion)
			componentsGroup.GET("/category/:category", a.Components.HandleListByCategory)
		}

		// Component routes - write operations (root only)
		componentsAdmin := v1.Group("/components")
		componentsAdmin.Use(a.rootAccessRequired())
		{
			componentsAdmin.POST("", a.Components.HandleCreate)
			componentsAdmin.PUT("/:id", a.Components.HandleUpdate)
			componentsAdmin.DELETE("/:id", a.Components.HandleDelete)
		}

		// Distribution downloads - read (auth required)
		distDownloadsRead := v1.Group("/distributions/:id/downloads")
		distDownloadsRead.Use(a.authRequired())
		{
			distDownloadsRead.GET("", a.Downloads.HandleListDistributionDownloads)
		}

		// Download job routes - read (auth required)
		downloadsRead := v1.Group("/downloads")
		downloadsRead.Use(a.authRequired())
		{
			downloadsRead.GET("/:jobId", a.Downloads.HandleGetDownloadJob)
		}

		// Download job routes - write (write access required)
		downloadsWrite := v1.Group("/downloads")
		downloadsWrite.Use(a.writeAccessRequired())
		{
			downloadsWrite.POST("/:jobId/cancel", a.Downloads.HandleCancelDownload)
			downloadsWrite.POST("/:jobId/retry", a.Downloads.HandleRetryDownload)
		}

		// Active downloads - admin only
		downloadsAdmin := v1.Group("/downloads")
		downloadsAdmin.Use(a.adminAccessRequired())
		{
			downloadsAdmin.GET("/active", a.Downloads.HandleListActiveDownloads)
		}
	}
}
