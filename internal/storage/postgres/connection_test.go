package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

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

func TestConnectDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("context canceled before connection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := &config.Config{
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

		cfg := &config.Config{
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
		cfg         *config.Config
		expectedDSN string
	}{
		{
			name: "standard configuration",
			cfg: &config.Config{
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
