package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab-engineering-metrics-api/internal/app"
	"gitlab-engineering-metrics-api/internal/config"
	"gitlab-engineering-metrics-api/internal/database"
	"gitlab-engineering-metrics-api/internal/observability"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Load config first to get log settings
	cfg := config.Load()

	// Create structured logger from config
	logger := observability.NewLogger(observability.Config{
		Format: cfg.LogFormat,
		Level:  cfg.LogLevel,
	})

	logger.Info("Starting GitLab Engineering Metrics API...")

	db, err := database.New(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("Database connection established")

	application := app.New(db, cfg, logger)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      application.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Server listening", slog.String("addr", cfg.ServerAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Server exited gracefully")
}



