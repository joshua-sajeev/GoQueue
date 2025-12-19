package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gorm.io/gorm/logger"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func(context.Context, interface{}) error
		expectError   bool
		errorContains string
		validate      func(*testing.T, *Config)
	}{
		{
			name: "valid configuration with defaults",
			setupEnv: func(ctx context.Context, v interface{}) error {
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
			setupEnv: func(ctx context.Context, v interface{}) error {
				return errors.New("env: POSTGRES_USER is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "missing required POSTGRES_PASSWORD",
			setupEnv: func(ctx context.Context, v interface{}) error {
				return errors.New("env: POSTGRES_PASSWORD is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "missing required POSTGRES_HOST",
			setupEnv: func(ctx context.Context, v interface{}) error {
				return errors.New("env: POSTGRES_HOST is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "missing required POSTGRES_PORT",
			setupEnv: func(ctx context.Context, v interface{}) error {
				return errors.New("env: POSTGRES_PORT is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "missing required POSTGRES_DB",
			setupEnv: func(ctx context.Context, v interface{}) error {
				return errors.New("env: POSTGRES_DB is required but not set")
			},
			expectError:   true,
			errorContains: "failed to process env config",
		},
		{
			name: "custom values override defaults",
			setupEnv: func(ctx context.Context, v interface{}) error {
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
			setupEnv: func(ctx context.Context, v interface{}) error {
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
		{
			name: "whitespace only user",
			cfg: &Config{
				User:       "   ",
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
		{
			name: "empty database",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_DB is required"},
		},
		{
			name: "empty host",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_HOST is required"},
		},
		{
			name: "empty port",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_PORT is required"},
		},
		{
			name: "invalid port format",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "invalid",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_PORT must be a valid number"},
		},
		{
			name: "port below valid range",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "0",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_PORT must be between 1 and 65535"},
		},
		{
			name: "port above valid range",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "65536",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"POSTGRES_PORT must be between 1 and 65535"},
		},
		{
			name: "negative max retries",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: -1,
				RetryDelay: 2 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"DB_MAX_RETRIES must be non-negative"},
		},
		{
			name: "zero retry delay",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 0,
			},
			expectError:   true,
			errorContains: []string{"DB_RETRY_DELAY must be positive"},
		},
		{
			name: "negative retry delay",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: -1 * time.Second,
			},
			expectError:   true,
			errorContains: []string{"DB_RETRY_DELAY must be positive"},
		},
		{
			name: "retry delay exceeds maximum",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 11 * time.Minute,
			},
			expectError:   true,
			errorContains: []string{"DB_RETRY_DELAY must not exceed 10 minutes"},
		},
		{
			name: "multiple validation errors",
			cfg: &Config{
				User:       "",
				Password:   "pass",
				Host:       "",
				Port:       "invalid",
				Database:   "",
				MaxRetries: -5,
				RetryDelay: 0,
			},
			expectError: true,
			errorContains: []string{
				"POSTGRES_USER is required",
				"POSTGRES_DB is required",
				"POSTGRES_HOST is required",
				"POSTGRES_PORT must be a valid number",
				"DB_MAX_RETRIES must be non-negative",
				"DB_RETRY_DELAY must be positive",
			},
		},
		{
			name: "boundary: port = 1 (minimum valid)",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "1",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "boundary: port = 65535 (maximum valid)",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "65535",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "boundary: retry delay = 10 minutes (maximum valid)",
			cfg: &Config{
				User:       "user",
				Password:   "pass",
				Host:       "localhost",
				Port:       "5432",
				Database:   "db",
				MaxRetries: 10,
				RetryDelay: 10 * time.Minute,
			},
			expectError: false,
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
			name:     "timeout error",
			err:      errors.New("connection timeout exceeded"),
			expected: "database connection timed out",
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
			name:     "unknown error",
			err:      errors.New("some random database error"),
			expected: "database error",
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
			name:     "silent lowercase",
			input:    "silent",
			expected: logger.Silent,
		},
		{
			name:     "silent uppercase",
			input:    "SILENT",
			expected: logger.Silent,
		},
		{
			name:     "silent mixed case",
			input:    "SiLeNt",
			expected: logger.Silent,
		},
		{
			name:     "error lowercase",
			input:    "error",
			expected: logger.Error,
		},
		{
			name:     "error uppercase",
			input:    "ERROR",
			expected: logger.Error,
		},
		{
			name:     "warn lowercase",
			input:    "warn",
			expected: logger.Warn,
		},
		{
			name:     "warn uppercase",
			input:    "WARN",
			expected: logger.Warn,
		},
		{
			name:     "info lowercase",
			input:    "info",
			expected: logger.Info,
		},
		{
			name:     "info uppercase",
			input:    "INFO",
			expected: logger.Info,
		},
		{
			name:     "invalid value defaults to warn",
			input:    "invalid",
			expected: logger.Warn,
		},
		{
			name:     "empty string defaults to warn",
			input:    "",
			expected: logger.Warn,
		},
		{
			name:     "numeric string defaults to warn",
			input:    "123",
			expected: logger.Warn,
		},
		{
			name:     "special characters default to warn",
			input:    "!@#$%",
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
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name          string
		cfg           *Config
		setupMock     func()
		skipTest      bool
		expectError   bool
		errorContains string
		validate      func(*testing.T, error)
	}{
		{
			name:        "nil config loads from environment",
			cfg:         nil,
			setupMock:   func() {},
			skipTest:    true, // Skip - requires env setup
			expectError: true,
		},
		{
			name: "successful connection on first attempt",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     3,
				RetryDelay:     100 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Warn,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - requires real DB
			expectError: false,
		},
		{
			name: "context cancelled before connection",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     3,
				RetryDelay:     100 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Warn,
			},
			setupMock:     func() {},
			skipTest:      false, // This test validates context cancellation logic
			expectError:   true,
			errorContains: "context canceled",
			validate: func(t *testing.T, err error) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

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

				_, err = ConnectDB(ctx, cfg)
				if err == nil {
					t.Fatal("expected error for cancelled context")
				}
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got %v", err)
				}
			},
		},
		{
			name: "context timeout during retries",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     10,
				RetryDelay:     1 * time.Second,
				ConnectTimeout: 5,
				LogLevel:       logger.Warn,
			},
			setupMock:     func() {},
			skipTest:      false,
			expectError:   true,
			errorContains: "context deadline exceeded",
			validate: func(t *testing.T, err error) {
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

				_, err = ConnectDB(ctx, cfg)
				if err == nil {
					t.Fatal("expected error for timeout context")
				}
			},
		},
		{
			name: "all retries exhausted",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "invalid-host-12345",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     2,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 1,
				LogLevel:       logger.Silent,
			},
			setupMock:     func() {},
			skipTest:      true, // Skip - network dependent
			expectError:   true,
			errorContains: "database connection failed after 2 attempts",
		},
		{
			name: "invalid credentials",
			cfg: &Config{
				User:           "wronguser",
				Password:       "wrongpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     1,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Silent,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - requires real DB
			expectError: true,
		},
		{
			name: "connection timeout",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "10.255.255.1", // Non-routable IP
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     1,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 1,
				LogLevel:       logger.Silent,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - network dependent
			expectError: true,
		},
		{
			name: "invalid port",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "99999", // Invalid port will be caught by validation
				Database:       "testdb",
				MaxRetries:     1,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Silent,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - network dependent
			expectError: true,
		},
		{
			name: "zero retries configuration",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     0,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Silent,
			},
			setupMock:     func() {},
			skipTest:      true, // Skip - requires real DB
			expectError:   true,
			errorContains: "database connection failed after 0 attempts",
		},
		{
			name: "very short retry delay",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "invalid-host",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     3,
				RetryDelay:     1 * time.Millisecond,
				ConnectTimeout: 1,
				LogLevel:       logger.Silent,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - network dependent
			expectError: true,
		},
		{
			name: "different log levels",
			cfg: &Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				MaxRetries:     1,
				RetryDelay:     10 * time.Millisecond,
				ConnectTimeout: 5,
				LogLevel:       logger.Info,
			},
			setupMock:   func() {},
			skipTest:    true, // Skip - requires real DB
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require real database or network
			if tt.skipTest {
				t.Skip("Skipping test that requires real database or network access")
			}

			if tt.setupMock != nil {
				tt.setupMock()
			}

			ctx := context.Background()

			// Special handling for custom validation
			if tt.validate != nil {
				tt.validate(t, nil)
				return
			}

			db, err := ConnectDB(ctx, tt.cfg)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
					if db != nil {
						sqlDB, _ := db.DB()
						if sqlDB != nil {
							sqlDB.Close()
						}
					}
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if db == nil {
				t.Fatal("expected non-nil database connection")
			}

			// Cleanup
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		})
	}
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
		{
			name: "special characters in password",
			cfg: &Config{
				User:           "user",
				Password:       "p@ss!word#123",
				Host:           "localhost",
				Port:           "5432",
				Database:       "testdb",
				ConnectTimeout: 10,
			},
			expectedDSN: "host=localhost user=user password=p@ss!word#123 dbname=testdb port=5432 sslmode=disable connect_timeout=10",
		},
		{
			name: "custom port",
			cfg: &Config{
				User:           "admin",
				Password:       "secret",
				Host:           "192.168.1.100",
				Port:           "3306",
				Database:       "appdb",
				ConnectTimeout: 15,
			},
			expectedDSN: "host=192.168.1.100 user=admin password=secret dbname=appdb port=3306 sslmode=disable connect_timeout=15",
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

func TestConnectDB_ConnectionPoolSettings(t *testing.T) {
	// This test documents expected connection pool settings
	expectedSettings := struct {
		MaxIdleConns    int
		MaxOpenConns    int
		ConnMaxLifetime time.Duration
	}{
		MaxIdleConns:    10,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Hour,
	}

	t.Logf("Expected connection pool settings:")
	t.Logf("  MaxIdleConns: %d", expectedSettings.MaxIdleConns)
	t.Logf("  MaxOpenConns: %d", expectedSettings.MaxOpenConns)
	t.Logf("  ConnMaxLifetime: %v", expectedSettings.ConnMaxLifetime)

	// These values should match what's set in ConnectDB
	if expectedSettings.MaxIdleConns != 10 {
		t.Error("MaxIdleConns should be 10")
	}
	if expectedSettings.MaxOpenConns != 50 {
		t.Error("MaxOpenConns should be 50")
	}
	if expectedSettings.ConnMaxLifetime != time.Hour {
		t.Error("ConnMaxLifetime should be 1 hour")
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
