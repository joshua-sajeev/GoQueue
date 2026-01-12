package config

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sethvargo/go-envconfig"
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
