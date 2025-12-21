package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	User           string        `env:"POSTGRES_USER,required"`
	Password       string        `env:"POSTGRES_PASSWORD,required"`
	Host           string        `env:"POSTGRES_HOST,required"`
	Port           string        `env:"POSTGRES_PORT,required"`
	Database       string        `env:"POSTGRES_DB,required"`
	MaxRetries     int           `env:"DB_MAX_RETRIES,default=10"`
	RetryDelay     time.Duration `env:"DB_RETRY_DELAY,default=2s"`
	ConnectTimeout int           `env:"DB_CONNECT_TIMEOUT,default=5"`
	LogLevelString string        `env:"DB_LOG_LEVEL,default=warn"`
	LogLevel       logger.LogLevel
}

// to help with testing
var envProcess = envconfig.Process

func LoadConfigFromEnv(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envProcess(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cfg.LogLevel = ParseLogLevel(cfg.LogLevelString)
	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	var errors []string

	if strings.TrimSpace(cfg.User) == "" {
		errors = append(errors, "POSTGRES_USER is required")
	}

	if strings.TrimSpace(cfg.Database) == "" {
		errors = append(errors, "POSTGRES_DB is required")
	}

	if strings.TrimSpace(cfg.Host) == "" {
		errors = append(errors, "POSTGRES_HOST is required")
	}

	if strings.TrimSpace(cfg.Port) == "" {
		errors = append(errors, "POSTGRES_PORT is required")
	}
	if cfg.Port != "" {
		port, err := strconv.Atoi(cfg.Port)
		if err != nil {
			errors = append(errors, "POSTGRES_PORT must be a valid number")
		} else if port < 1 || port > 65535 {
			errors = append(errors, "POSTGRES_PORT must be between 1 and 65535")
		}
	}

	if cfg.MaxRetries < 0 {
		errors = append(errors, "DB_MAX_RETRIES must be non-negative")
	}

	if cfg.RetryDelay <= 0 {
		errors = append(errors, "DB_RETRY_DELAY must be positive")
	}

	if cfg.RetryDelay > 10*time.Minute {
		errors = append(errors, "DB_RETRY_DELAY must not exceed 10 minutes")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// ConnectDB establishes connection to PostgreSQL with context support
func ConnectDB(ctx context.Context, cfg *Config) (*gorm.DB, error) {
	if cfg == nil {
		loadedCfg, err := LoadConfigFromEnv(ctx)
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable connect_timeout=%d",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.ConnectTimeout,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
	}

	// Try connection with retries
	for i := 0; i < cfg.MaxRetries; i++ {
		// Check if context is cancelled before attempting connection
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		gdb, err := gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			sqlDB, dbErr := gdb.DB()
			if dbErr == nil {
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				pingErr := sqlDB.PingContext(pingCtx)
				cancel()

				if pingErr == nil {

					sqlDB.SetMaxIdleConns(10)
					sqlDB.SetMaxOpenConns(50)
					sqlDB.SetConnMaxLifetime(time.Hour)

					return gdb, nil
				}
				err = pingErr
			} else {
				err = dbErr
			}
		}

		// Respect context cancellation during retry delay
		select {
		case <-time.After(cfg.RetryDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("database connection failed after %d attempts", cfg.MaxRetries)
}

// simplifyDBError returns a user-friendly error message
func simplifyDBError(err error) string {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "password authentication failed"):
		return "invalid database credentials"
	case strings.Contains(msg, "timeout"):
		return "database connection timed out"
	case strings.Contains(msg, "connect"):
		return "cannot reach database server"
	case strings.Contains(msg, "SASL"):
		return "authentication error"
	}

	return "database error"
}

// Convert string to logger.LogLevel
func ParseLogLevel(levelStr string) logger.LogLevel {
	switch strings.ToLower(levelStr) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
