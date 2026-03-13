package observability

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
)

// Config holds logger configuration
type Config struct {
	Format  string // "json" or "text"
	Level   string // "debug", "info", "warn", "error"
	LogFile string // path to log file, empty for stdout only
}

// NewLogger creates a new structured logger with the specified configuration
func NewLogger(config Config) *slog.Logger {
	// Determine log level
	level := parseLogLevel(config.Level)

	// Determine format
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create multi-writer: both stdout and file (if configured)
	writers := []io.Writer{os.Stdout}

	if config.LogFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.LogFile)
		if logDir != "." && logDir != "" {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			} else {
				// Open log file for append
				file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
				} else {
					writers = append(writers, file)
					// Log startup message to file
					fmt.Fprintf(file, "\n--- Log started at %s ---\n", time.Now().Format(time.RFC3339))
				}
			}
		}
	}

	multiWriter := io.MultiWriter(writers...)

	var handler slog.Handler
	if strings.ToLower(config.Format) == "text" {
		handler = slog.NewTextHandler(multiWriter, opts)
	} else {
		handler = slog.NewJSONHandler(multiWriter, opts)
	}

	return slog.New(handler)
}

// GetLogger returns a singleton logger instance
func GetLogger() *slog.Logger {
	loggerOnce.Do(func() {
		config := Config{
			Format:  getEnv("LOG_FORMAT", "json"),
			Level:   getEnv("LOG_LEVEL", "info"),
			LogFile: getEnv("LOG_FILE", "logs/app.log"),
		}
		logger = NewLogger(config)
	})
	return logger
}

// SetLogger sets the global logger instance (useful for testing)
func SetLogger(l *slog.Logger) {
	logger = l
}

// parseLogLevel converts a string level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getEnv retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
