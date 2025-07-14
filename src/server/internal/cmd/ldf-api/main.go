package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var logger *log.Logger

func init() {
	// Initialize logger
	logger = log.New(os.Stderr)
	
	// Set up configuration
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.read_timeout", "10s")
	viper.SetDefault("api.write_timeout", "10s")
	viper.SetDefault("api.cors_enabled", true)
	viper.SetDefault("log.level", "info")
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/ldf/")
	viper.AddConfigPath("$HOME/.config/ldf")
	viper.AddConfigPath(".")
	
	viper.AutomaticEnv()
	viper.SetEnvPrefix("LDF")
	
	if err := viper.ReadInConfig(); err != nil {
		logger.Info("No config file found, using defaults")
	} else {
		logger.Info("Using config file", "file", viper.ConfigFileUsed())
	}
	
	// Set log level
	switch viper.GetString("log.level") {
	case "debug":
		logger.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
	case "warn":
		logger.SetLevel(log.WarnLevel)
		gin.SetMode(gin.ReleaseMode)
	case "error":
		logger.SetLevel(log.ErrorLevel)
		gin.SetMode(gin.ReleaseMode)
	default:
		logger.SetLevel(log.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	// Create Gin router
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(ginLogger())
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC(),
		})
	})
	
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version": "1.0.0",
				"api":     "v1",
			})
		})
		
		// TODO: Add more routes here
		// distributions := v1.Group("/distributions")
		// boards := v1.Group("/boards")
		// kernels := v1.Group("/kernels")
	}
	
	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", viper.GetString("api.host"), viper.GetInt("api.port"))
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  parseDuration(viper.GetString("api.read_timeout")),
		WriteTimeout: parseDuration(viper.GetString("api.write_timeout")),
	}
	
	// Start server in goroutine
	go func() {
		logger.Info("Starting API server", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}
	
	logger.Info("Server exited")
}

func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// Process request
		c.Next()
		
		// Log request details
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		
		if raw != "" {
			path = path + "?" + raw
		}
		
		switch {
		case statusCode >= 500:
			logger.Error("Server error",
				"status", statusCode,
				"method", method,
				"path", path,
				"ip", clientIP,
				"latency", latency,
			)
		case statusCode >= 400:
			logger.Warn("Client error",
				"status", statusCode,
				"method", method,
				"path", path,
				"ip", clientIP,
				"latency", latency,
			)
		default:
			logger.Info("Request",
				"status", statusCode,
				"method", method,
				"path", path,
				"ip", clientIP,
				"latency", latency,
			)
		}
	}
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		logger.Warn("Invalid duration, using default", "duration", s, "default", "10s")
		return 10 * time.Second
	}
	return d
}
