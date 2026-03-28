package config

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	ServerAddr        string
	MaxOpenConns      int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	ClientCredentials map[string]string
	LogFormat         string
	LogLevel          string
}

// Load reads configuration from environment variables
func Load() *Config {
	maxOpenConns := getEnvAsInt("MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvAsInt("MAX_IDLE_CONNS", 5)
	connMaxLifetime := getEnvAsDuration("CONN_MAX_LIFETIME", 5*time.Minute)

	cfg := &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBName:            getEnv("DB_NAME", "gitlab_metrics"),
		DBUser:            getEnv("DB_USER", ""),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		ServerAddr:        getEnv("SERVER_ADDR", ":8080"),
		MaxOpenConns:      maxOpenConns,
		MaxIdleConns:      maxIdleConns,
		ConnMaxLifetime:   connMaxLifetime,
		ClientCredentials: make(map[string]string),
		LogFormat:         getEnv("LOG_FORMAT", "json"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}

	// Load client credentials from environment variable
	// Format: "client1:secret1,client2:secret2"
	cfg.loadClientCredentials()

	return cfg
}

// GetDSN returns the PostgreSQL DSN string
func (c *Config) GetDSN() string {
	q := url.Values{}
	q.Set("sslmode", c.DBSSLMode)
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     c.DBHost + ":" + c.DBPort,
		Path:     "/" + c.DBName,
		RawQuery: q.Encode(),
	}
	return u.String()
}

// loadClientCredentials parses client credentials from environment variable
func (c *Config) loadClientCredentials() {
	credStr := getEnv("CLIENT_CREDENTIALS", "")
	if credStr == "" {
		return
	}

	pairs := strings.Split(credStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}

		client := strings.TrimSpace(parts[0])
		secret := strings.TrimSpace(parts[1])
		if client != "" && secret != "" {
			c.ClientCredentials[client] = secret
		}
	}
}

// GetMaxOpenConns returns the maximum number of open database connections
func (c *Config) GetMaxOpenConns() int {
	return c.MaxOpenConns
}

// GetMaxIdleConns returns the maximum number of idle database connections
func (c *Config) GetMaxIdleConns() int {
	return c.MaxIdleConns
}

// GetConnMaxLifetime returns the maximum lifetime of a database connection
func (c *Config) GetConnMaxLifetime() time.Duration {
	return c.ConnMaxLifetime
}

// getEnv retrieves an environment variable with a default value.
// If the direct env var is empty, it checks for <KEY>_FILE and reads the value from the file.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	fileKey := key + "_FILE"
	if filePath := os.Getenv(fileKey); filePath != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: failed to read %s from %s: %v", fileKey, filePath, err)
			return defaultValue
		}

		valueFromFile := strings.TrimSpace(string(b))
		if valueFromFile != "" {
			return valueFromFile
		}
	}

	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: invalid value for %s: %q, using default %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsDuration retrieves an environment variable as a duration with a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: invalid value for %s: %q, using default %v", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
