package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/internal/config"
	"github.com/amitbasuri/taskqueue-runner-go/internal/storage/postgres"
	"github.com/amitbasuri/taskqueue-runner-go/internal/worker"
	"github.com/amitbasuri/taskqueue-runner-go/internal/worker/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Load the dotenv if exists
	_ = godotenv.Load()

	var env config.Worker
	err := envconfig.Process("", &env)
	if err != nil {
		log.Fatal("Cannot load env:", err)
	}

	// Setup structured logging
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	slog.Info("Starting Task Queue Worker (Consumer)")

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

	// Initialize handler registry with task handlers
	handlerRegistry := worker.NewHandlerRegistry()
	handlerRegistry.Register(handlers.NewSendEmailHandler())
	handlerRegistry.Register(handlers.NewRunQueryHandler())

	slog.Info("Registered task handlers", "handlers", handlerRegistry.List())

	// Start worker
	workerConfig := worker.Config{
		PollInterval: time.Duration(env.PollInterval) * time.Second,
		TaskTimeout:  time.Duration(env.TaskTimeout) * time.Second,
	}
	w := worker.NewWorker(store, handlerRegistry, workerConfig)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w.Start(ctx)
	slog.Info("Worker stopped gracefully")
}
