package observability

import (
	"log/slog"
	"os"
	"sync"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool // just check it doesn't panic
	}{
		{
			name: "json format with info level",
			config: Config{
				Format: "json",
				Level:  "info",
			},
			want: true,
		},
		{
			name: "text format with debug level",
			config: Config{
				Format: "text",
				Level:  "debug",
			},
			want: true,
		},
		{
			name: "json format with warn level",
			config: Config{
				Format: "json",
				Level:  "warn",
			},
			want: true,
		},
		{
			name: "json format with error level",
			config: Config{
				Format: "json",
				Level:  "error",
			},
			want: true,
		},
		{
			name: "uppercase format and level",
			config: Config{
				Format: "JSON",
				Level:  "INFO",
			},
			want: true,
		},
		{
			name: "warning alias for warn",
			config: Config{
				Format: "text",
				Level:  "warning",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			logger := NewLogger(tt.config)
			if logger == nil {
				t.Error("NewLogger() returned nil")
			}

			// Test that we can log without panic
			logger.Info("test message", slog.String("key", "value"))
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"warning", "warning", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"unknown", "unknown", slog.LevelInfo},
		{"empty", "", slog.LevelInfo},
		{"uppercase DEBUG", "DEBUG", slog.LevelDebug},
		{"uppercase INFO", "INFO", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.level)
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestGetLogger_Singleton(t *testing.T) {
	// Reset the logger to test singleton behavior
	logger = nil
	loggerOnce = sync.Once{}

	// First call should create the logger
	l1 := GetLogger()
	if l1 == nil {
		t.Fatal("GetLogger() returned nil")
	}

	// Second call should return the same instance
	l2 := GetLogger()
	if l2 != l1 {
		t.Error("GetLogger() should return the same instance")
	}
}

func TestSetLogger(t *testing.T) {
	// Create a custom logger
	customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Set it as the global logger
	SetLogger(customLogger)

	// Get the logger and verify it's our custom one
	got := GetLogger()
	if got != customLogger {
		t.Error("SetLogger() did not set the logger correctly")
	}

	// Reset for other tests
	logger = nil
	loggerOnce = sync.Once{}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_LOGGER_ENV", "test_value")
	defer os.Unsetenv("TEST_LOGGER_ENV")

	got := getEnv("TEST_LOGGER_ENV", "default")
	if got != "test_value" {
		t.Errorf("getEnv() = %v, want %v", got, "test_value")
	}

	// Test with non-existing env var
	got = getEnv("NON_EXISTENT_ENV_VAR", "default")
	if got != "default" {
		t.Errorf("getEnv() = %v, want %v", got, "default")
	}
}
