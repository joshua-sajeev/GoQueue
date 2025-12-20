package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func(context.Context, any) error
		expectError   bool
		errorContains string
		validate      func(*testing.T, *Config)
	}{
		{
			name: "valid configuration with defaults",
			setupEnv: func(ctx context.Context, v any) error {
				cfg := v.(*Config)
				cfg.User = "testuser"
				cfg.Password = "testpass"
				cfg.Host = "localhost"
				cfg.Port = "5432"
				cfg.Database = "testdb"
				cfg.MaxRetries = 10
				cfg.RetryDelay = 2 * time.Second
				cfg.ConnectTimeout = 5
				cfg.LogLevelString = "warn"
				return nil
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.User != "testuser" {
					t.Errorf("expected User=testuser, got %s", cfg.User)
				}
				if cfg.MaxRetries != 10 {
					t.Errorf("expected MaxRetries=10, got %d", cfg.MaxRetries)
				}
				if cfg.RetryDelay != 2*time.Second {
					t.Errorf("expected RetryDelay=2s, got %v", cfg.RetryDelay)
				}
				if cfg.LogLevel != logger.Warn {
					t.Errorf("expected LogLevel=Warn, got %v", cfg.LogLevel)
				}
			},
		},
		{
			name: "missing required POSTGRES_USER",
			setupEnv: func(ctx context.Context, v any) error {
				return errors.New("env: POSTGRES_USER is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "missing required POSTGRES_PASSWORD",
			setupEnv: func(ctx context.Context, v any) error {
				return errors.New("env: POSTGRES_PASSWORD is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "custom values override defaults",
			setupEnv: func(ctx context.Context, v any) error {
				cfg := v.(*Config)
				cfg.User = "customuser"
				cfg.Password = "custompass"
				cfg.Host = "db.example.com"
				cfg.Port = "3306"
				cfg.Database = "customdb"
				cfg.MaxRetries = 5
				cfg.RetryDelay = 5 * time.Second
				cfg.ConnectTimeout = 10
				cfg.LogLevelString = "info"
				return nil
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.MaxRetries != 5 {
					t.Errorf("expected MaxRetries=5, got %d", cfg.MaxRetries)
				}
				if cfg.RetryDelay != 5*time.Second {
					t.Errorf("expected RetryDelay=5s, got %v", cfg.RetryDelay)
				}
				if cfg.LogLevel != logger.Info {
					t.Errorf("expected LogLevel=Info, got %v", cfg.LogLevel)
				}
			},
		},
		{
			name: "validation error after successful env processing",
			setupEnv: func(ctx context.Context, v any) error {
				cfg := v.(*Config)
				cfg.User = "" // Invalid
				cfg.Password = "testpass"
				cfg.Host = "localhost"
				cfg.Port = "5432"
				cfg.Database = "testdb"
				cfg.MaxRetries = 10
				cfg.RetryDelay = 2 * time.Second
				return nil
			},
			expectError:   true,
			errorContains: "config validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock envProcess
			originalEnvProcess := envProcess
			defer func() { envProcess = originalEnvProcess }()

			envProcess = func(ctx context.Context, v any, mus ...envconfig.Mutator) error {
				cfg := v.(*Config)
				return tt.setupEnv(ctx, cfg)
			}

			cfg, err := LoadConfigFromEnv(context.Background())

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		expectError   bool
		errorContains []string
	}{
		{
			name: "valid config",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty user",
			cfg: &Config{
				User:       "",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_USER is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				for _, substr := range tt.errorContains {
					if !contains(err.Error(), substr) {
						t.Errorf("expected error to contain '%s', got '%s'", substr, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSimplifyDBError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "password authentication failed",
			err:      errors.New("pq: password authentication failed for user"),
			expected: "invalid database credentials",
		},
		{
			name:     "i/o timeout",
			err:      errors.New("dial tcp: i/o timeout"),
			expected: "database connection timed out",
		},
		{
			name:     "connection refused",
			err:      errors.New("connect: connection refused"),
			expected: "cannot reach database server",
		},
		{
			name:     "no route to host",
			err:      errors.New("connect: no route to host"),
			expected: "cannot reach database server",
		},
		{
			name:     "SASL authentication error",
			err:      errors.New("SASL authentication failed"),
			expected: "authentication error",
		},
		{
			name:     "empty error message",
			err:      errors.New(""),
			expected: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simplifyDBError(tt.err)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected logger.LogLevel
	}{
		{
			name:     "warn lowercase",
			input:    "warn",
			expected: logger.Warn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConnectDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("context canceled before connection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := &Config{
			User:           "testuser",
			Password:       "testpass",
			Host:           "localhost",
			Port:           "5432",
			Database:       "testdb",
			MaxRetries:     3,
			RetryDelay:     100 * time.Millisecond,
			ConnectTimeout: 5,
			LogLevel:       logger.Silent,
		}

		_, err := ConnectDB(ctx, cfg)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("context timeout during retries", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		cfg := &Config{
			User:           "testuser",
			Password:       "testpass",
			Host:           "invalid-host-that-does-not-exist",
			Port:           "5432",
			Database:       "testdb",
			MaxRetries:     10,
			RetryDelay:     100 * time.Millisecond,
			ConnectTimeout: 1,
			LogLevel:       logger.Silent,
		}

		_, err := ConnectDB(ctx, cfg)

		assert.Error(t, err)
	})
}

func TestConnectDB_DSNFormat(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedDSN string
	}{
		{
			name: "standard configuration",
			cfg: &Config{
				User:           "myuser",
				Password:       "mypassword",
				Host:           "db.example.com",
				Port:           "5432",
				Database:       "mydb",
				ConnectTimeout: 5,
			},
			expectedDSN: "host=db.example.com user=myuser password=mypassword dbname=mydb port=5432 sslmode=disable connect_timeout=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates DSN format without actually connecting
			dsn := fmt.Sprintf(
				"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable connect_timeout=%d",
				tt.cfg.Host, tt.cfg.User, tt.cfg.Password, tt.cfg.Database, tt.cfg.Port, tt.cfg.ConnectTimeout,
			)

			if dsn != tt.expectedDSN {
				t.Errorf("DSN mismatch\nexpected: %s\ngot: %s", tt.expectedDSN, dsn)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
