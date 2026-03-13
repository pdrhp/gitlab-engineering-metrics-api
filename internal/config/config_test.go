package config

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected *Config
	}{
		{
			name: "default values",
			env:  map[string]string{},
			expected: &Config{
				DBHost:            "localhost",
				DBPort:            "5432",
				DBName:            "gitlab_metrics",
				DBUser:            "",
				DBPassword:        "",
				DBSSLMode:         "disable",
				ServerAddr:        ":8080",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnMaxLifetime:   5 * time.Minute,
				ClientCredentials: map[string]string{},
			},
		},
		{
			name: "custom database values",
			env: map[string]string{
				"DB_HOST":     "postgres.example.com",
				"DB_PORT":     "5433",
				"DB_NAME":     "custom_db",
				"DB_USER":     "app_user",
				"DB_PASSWORD": "secret123",
				"DB_SSLMODE":  "require",
			},
			expected: &Config{
				DBHost:            "postgres.example.com",
				DBPort:            "5433",
				DBName:            "custom_db",
				DBUser:            "app_user",
				DBPassword:        "secret123",
				DBSSLMode:         "require",
				ServerAddr:        ":8080",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnMaxLifetime:   5 * time.Minute,
				ClientCredentials: map[string]string{},
			},
		},
		{
			name: "custom server address",
			env: map[string]string{
				"SERVER_ADDR": ":3000",
			},
			expected: &Config{
				DBHost:            "localhost",
				DBPort:            "5432",
				DBName:            "gitlab_metrics",
				DBUser:            "",
				DBPassword:        "",
				DBSSLMode:         "disable",
				ServerAddr:        ":3000",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnMaxLifetime:   5 * time.Minute,
				ClientCredentials: map[string]string{},
			},
		},
		{
			name: "all custom values",
			env: map[string]string{
				"DB_HOST":           "prod.db.com",
				"DB_PORT":           "5432",
				"DB_NAME":           "metrics_prod",
				"DB_USER":           "metrics_app",
				"DB_PASSWORD":       "secure_pass",
				"DB_SSLMODE":        "require",
				"SERVER_ADDR":       ":9090",
				"MAX_OPEN_CONNS":    "50",
				"MAX_IDLE_CONNS":    "10",
				"CONN_MAX_LIFETIME": "10m",
			},
			expected: &Config{
				DBHost:            "prod.db.com",
				DBPort:            "5432",
				DBName:            "metrics_prod",
				DBUser:            "metrics_app",
				DBPassword:        "secure_pass",
				DBSSLMode:         "require",
				ServerAddr:        ":9090",
				MaxOpenConns:      50,
				MaxIdleConns:      10,
				ConnMaxLifetime:   10 * time.Minute,
				ClientCredentials: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variables using t.Setenv() for automatic cleanup
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			// Load config
			config := Load()

			// Verify values
			if config.DBHost != tt.expected.DBHost {
				t.Errorf("DBHost = %v, want %v", config.DBHost, tt.expected.DBHost)
			}
			if config.DBPort != tt.expected.DBPort {
				t.Errorf("DBPort = %v, want %v", config.DBPort, tt.expected.DBPort)
			}
			if config.DBName != tt.expected.DBName {
				t.Errorf("DBName = %v, want %v", config.DBName, tt.expected.DBName)
			}
			if config.DBUser != tt.expected.DBUser {
				t.Errorf("DBUser = %v, want %v", config.DBUser, tt.expected.DBUser)
			}
			if config.DBPassword != tt.expected.DBPassword {
				t.Errorf("DBPassword = %v, want %v", config.DBPassword, tt.expected.DBPassword)
			}
			if config.DBSSLMode != tt.expected.DBSSLMode {
				t.Errorf("DBSSLMode = %v, want %v", config.DBSSLMode, tt.expected.DBSSLMode)
			}
			if config.ServerAddr != tt.expected.ServerAddr {
				t.Errorf("ServerAddr = %v, want %v", config.ServerAddr, tt.expected.ServerAddr)
			}
			if config.MaxOpenConns != tt.expected.MaxOpenConns {
				t.Errorf("MaxOpenConns = %v, want %v", config.MaxOpenConns, tt.expected.MaxOpenConns)
			}
			if config.MaxIdleConns != tt.expected.MaxIdleConns {
				t.Errorf("MaxIdleConns = %v, want %v", config.MaxIdleConns, tt.expected.MaxIdleConns)
			}
			if config.ConnMaxLifetime != tt.expected.ConnMaxLifetime {
				t.Errorf("ConnMaxLifetime = %v, want %v", config.ConnMaxLifetime, tt.expected.ConnMaxLifetime)
			}
		})
	}
}

func TestGetDSN(t *testing.T) {
	config := &Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "test_db",
		DBUser:     "test_user",
		DBPassword: "test_pass",
		DBSSLMode:  "disable",
	}

	dsn := config.GetDSN()
	expected := "postgres://test_user:test_pass@localhost:5432/test_db?sslmode=disable"

	if dsn != expected {
		t.Errorf("GetDSN() = %v, want %v", dsn, expected)
	}
}

func TestLoadClientCredentials(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected map[string]string
	}{
		{
			name:     "no credentials",
			env:      map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "single credential",
			env: map[string]string{
				"CLIENT_CREDENTIALS": "client1:secret1",
			},
			expected: map[string]string{
				"client1": "secret1",
			},
		},
		{
			name: "multiple credentials",
			env: map[string]string{
				"CLIENT_CREDENTIALS": "client1:secret1,client2:secret2,client3:secret3",
			},
			expected: map[string]string{
				"client1": "secret1",
				"client2": "secret2",
				"client3": "secret3",
			},
		},
		{
			name: "credentials with spaces",
			env: map[string]string{
				"CLIENT_CREDENTIALS": " client1 : secret1 , client2 : secret2 ",
			},
			expected: map[string]string{
				"client1": "secret1",
				"client2": "secret2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variables using t.Setenv() for automatic cleanup
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			// Load config
			config := Load()

			// Verify credentials
			if len(config.ClientCredentials) != len(tt.expected) {
				t.Errorf("ClientCredentials length = %v, want %v", len(config.ClientCredentials), len(tt.expected))
			}

			for client, secret := range tt.expected {
				if config.ClientCredentials[client] != secret {
					t.Errorf("ClientCredentials[%s] = %v, want %v", client, config.ClientCredentials[client], secret)
				}
			}
		})
	}
}
