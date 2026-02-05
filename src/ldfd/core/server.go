package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/api"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/db/migrations"
	_ "github.com/bitswalk/ldf/src/ldfd/docs"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/forge"
	"github.com/bitswalk/ldf/src/ldfd/storage"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server holds the HTTP server instance and configuration
type Server struct {
	router          *gin.Engine
	httpServer      *http.Server
	database        *db.Database
	storage         storage.Backend
	downloadManager *download.Manager
	buildManager    *build.Manager
	api             *api.API
}

// NewServer creates a new Server instance
func NewServer(database *db.Database, storageBackend storage.Backend) *Server {
	// Set Gin mode based on log level
	if viper.GetString("log.level") == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Add logging middleware
	router.Use(ginLogger())

	// Initialize auth components
	userManager := auth.NewUserManager(database.DB())
	jwtCfg := auth.DefaultJWTConfig()
	jwtService := auth.NewJWTService(jwtCfg, userManager, database)

	// Initialize download manager
	download.SetLogger(log)
	downloadCfg := download.DefaultConfig()
	downloadManager := download.NewManager(database, storageBackend, downloadCfg)

	// Initialize source version repository and discovery service
	sourceVersionRepo := db.NewSourceVersionRepository(database)
	componentRepo := db.NewComponentRepository(database)
	sourceRepo := db.NewSourceRepository(database)
	versionDiscovery := download.NewVersionDiscovery(sourceVersionRepo, componentRepo, sourceRepo)

	// Initialize build manager
	build.SetLogger(log)
	buildCfg := build.DefaultConfig()
	buildManager := build.NewManager(database, storageBackend, downloadManager, buildCfg)

	// Initialize forge registry for source detection and defaults
	forge.SetLogger(log)
	forgeRegistry := forge.NewRegistry()

	// Create API instance with all dependencies
	api.SetLogger(log)
	api.SetVersionInfo(VersionInfo)
	apiInstance := api.New(api.Config{
		DistRepo:          db.NewDistributionRepository(database),
		SourceRepo:        sourceRepo,
		ComponentRepo:     componentRepo,
		SourceVersionRepo: sourceVersionRepo,
		DownloadJobRepo:   db.NewDownloadJobRepository(database),
		LangPackRepo:      db.NewLanguagePackRepository(database),
		Database:          database,
		Storage:           storageBackend,
		UserManager:       userManager,
		JWTService:        jwtService,
		DownloadManager:   downloadManager,
		BuildManager:      buildManager,
		VersionDiscovery:  versionDiscovery,
		ForgeRegistry:     forgeRegistry,
	})

	// Register all routes
	apiInstance.RegisterRoutes(router)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	s := &Server{
		router:          router,
		database:        database,
		storage:         storageBackend,
		downloadManager: downloadManager,
		buildManager:    buildManager,
		api:             apiInstance,
	}

	// Start download manager
	go func() {
		if err := downloadManager.Start(context.Background()); err != nil {
			log.Error("Failed to start download manager", "error", err)
		}
	}()

	// Start build manager
	go func() {
		if err := buildManager.Start(context.Background()); err != nil {
			log.Error("Failed to start build manager", "error", err)
		}
	}()

	// Start version sync for all sources at startup
	// This refreshes the upstream versions cache
	go func() {
		// Small delay to allow server to fully initialize
		time.Sleep(2 * time.Second)
		log.Info("Initiating startup version discovery for all sources")
		versionDiscovery.SyncAllSources(context.Background(), sourceRepo)
	}()

	return s
}

// Run starts the HTTP server
func (s *Server) Run() error {
	bind := viper.GetString("server.bind")
	port := viper.GetInt("server.port")
	addr := fmt.Sprintf("%s:%d", bind, port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors coming from the listener
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		log.Info("Starting ldfd server", "address", addr)

		if s.storage != nil {
			log.Info("Storage enabled", "type", s.storage.Type(), "location", s.storage.Location())
		} else {
			log.Warn("Storage not configured - artifact endpoints disabled")
		}
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		log.Info("Received signal, shutting down", "signal", sig)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	log.Info("Server stopped gracefully")
	return nil
}

// Shutdown performs a graceful shutdown of the server and database
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop build manager
	if s.buildManager != nil {
		log.Info("Stopping build manager")
		if err := s.buildManager.Stop(); err != nil {
			log.Error("Build manager shutdown error", "error", err)
		}
	}

	// Stop download manager
	if s.downloadManager != nil {
		log.Info("Stopping download manager")
		if err := s.downloadManager.Stop(); err != nil {
			log.Error("Download manager shutdown error", "error", err)
		}
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Error("HTTP server shutdown error", "error", err)
		}
	}

	// Persist and close database
	if s.database != nil {
		log.Info("Persisting database to disk")
		if err := s.database.Shutdown(); err != nil {
			log.Error("Database shutdown error", "error", err)
			return err
		}
		log.Info("Database persisted successfully")
	}

	return nil
}

// corsMiddleware returns a gin middleware for handling CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow all origins for now (can be restricted via config later)
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Subject-Token")
			c.Header("Access-Control-Expose-Headers", "X-Subject-Token")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ginLogger returns a gin middleware for logging requests
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		log.Debug("HTTP request",
			"status", status,
			"method", method,
			"path", path,
			"latency", latency,
			"client_ip", c.ClientIP(),
		)
	}
}

// runServer is called by the root command to start the server
func runServer() error {
	log.Info("ldfd starting",
		"version", VersionInfo.Version,
		"build_date", VersionInfo.BuildDate,
		"log_output", log.Output(),
	)

	// Initialize database
	dbPath := viper.GetString("database.path")
	log.Info("Initializing database", "persist_path", dbPath)

	// Set logger for migrations before initializing database
	migrations.SetLogger(log)

	database, err := db.New(db.Config{
		PersistPath: dbPath,
		LoadOnStart: true,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load settings from database - these have highest priority
	// and override CLI/config file values
	log.Info("Loading configuration from database")
	if err := api.LoadConfigFromDatabase(database); err != nil {
		log.Warn("Failed to load configuration from database", "error", err)
	}

	// Sync any new settings that don't exist in database yet
	// This ensures new config options are persisted without overwriting user values
	log.Debug("Syncing missing settings to database")
	if err := api.SyncConfigToDatabase(database); err != nil {
		log.Warn("Failed to sync configuration to database", "error", err)
	}

	// Initialize storage backend
	var storageBackend storage.Backend
	storageType := viper.GetString("storage.type")

	// If S3 endpoint is specified, use S3 regardless of storage.type
	s3Endpoint := viper.GetString("storage.s3.endpoint")
	if s3Endpoint != "" {
		storageType = "s3"
	}

	log.Info("Initializing storage", "type", storageType)

	storageCfg := storage.Config{
		Type: storageType,
		Local: storage.LocalConfig{
			BasePath: viper.GetString("storage.local.path"),
		},
		S3: storage.S3Config{
			Provider:        storage.S3Provider(viper.GetString("storage.s3.provider")),
			Endpoint:        s3Endpoint,
			Region:          viper.GetString("storage.s3.region"),
			Bucket:          viper.GetString("storage.s3.bucket"),
			AccessKeyID:     viper.GetString("storage.s3.access_key"),
			SecretAccessKey: viper.GetString("storage.s3.secret_key"),
		},
	}

	storageBackend, err = storage.New(storageCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// For S3 backend, ensure bucket exists
	if s3Backend, ok := storageBackend.(*storage.S3Backend); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s3Backend.EnsureBucket(ctx); err != nil {
			log.Warn("S3 bucket not accessible - artifacts may not work correctly", "bucket", s3Backend.Bucket(), "error", err)
		} else {
			log.Debug("S3 bucket verified", "bucket", s3Backend.Bucket())
		}
	}

	server := NewServer(database, storageBackend)

	// Run server (blocks until shutdown signal)
	err = server.Run()

	// Ensure database is persisted on shutdown
	log.Info("Persisting database to disk")
	if dbErr := database.Shutdown(); dbErr != nil {
		log.Error("Failed to persist database", "error", dbErr)
		if err == nil {
			err = dbErr
		}
	} else {
		log.Info("Database persisted successfully")
	}

	return err
}
