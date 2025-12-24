package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/db"
	"github.com/amitbasuri/taskqueue-runner-go/internal/api"
	"github.com/amitbasuri/taskqueue-runner-go/internal/config"
	"github.com/amitbasuri/taskqueue-runner-go/internal/storage/postgres"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Load the dotenv if exists
	_ = godotenv.Load()

	var env config.Server
	err := envconfig.Process("", &env)
	if err != nil {
		log.Fatal("Cannot load env:", err)
	}

	// Setup structured logging
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	slog.Info("Starting Task Queue API Server (Producer)")

	// Run database migrations
	d, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		log.Fatal("Failed to load migrations:", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, env.Database.ToMigrationUri())
	if err != nil {
		log.Fatal("Failed to create migrate instance:", err)
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal("Failed to run migrations:", err)
		}
	}
	slog.Info("Migrations ran successfully")

	// Initialize database connection pool
	dbPool, err := pgxpool.New(context.Background(), env.Database.ToDbConnectionUri())
	if err != nil {
		log.Fatal("Failed to create database pool:", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	slog.Info("Database connection established")

	// Initialize storage layer
	store := postgres.NewStore(dbPool)

	// Initialize API handler
	apiHandler := api.NewHandler(store)

	// Setup HTTP routes
	r := gin.Default()

	// Register API routes
	apiHandler.RegisterRoutes(r)

	// Health check endpoints
	r.GET("/readiness", func(c *gin.Context) {
		// Check database connection
		if err := dbPool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/liveness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})

	// Task API endpoints
	r.POST("/tasks", apiHandler.CreateTask)
	r.GET("/tasks/:id", apiHandler.GetTask)
	r.GET("/tasks/:id/history", apiHandler.GetTaskHistory)

	srv := &http.Server{
		Addr:    ":" + env.ServerPort,
		Handler: r,
	}

	// Start HTTP server in goroutine
	go func() {
		slog.Info("HTTP server listening", "port", env.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down API server...")

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	slog.Info("API server exited gracefully")
}
