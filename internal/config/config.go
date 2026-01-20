package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
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
	ServerPort     string        `env:"SERVER_PORT,default=8080"`
	MaxWorkers     int           `env:"MAX_WORKERS,default=10"`
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
