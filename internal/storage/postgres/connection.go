package postgres

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	User           string        `env:"POSTGRES_USER,default=postgres"`
	Password       string        `env:"POSTGRES_PASSWORD,default=postgres"`
	Host           string        `env:"POSTGRES_HOST,default=postgres"`
	Port           string        `env:"POSTGRES_PORT,default=5432"`
	Database       string        `env:"POSTGRES_DB,default=taskdb"`
	MaxRetries     int           `env:"DB_MAX_RETRIES,default=10"`
	RetryDelay     time.Duration `env:"DB_RETRY_DELAY,default=2s"`
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

	// Validate required fields
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
	// Validate port is numeric and in valid range
	if cfg.Port != "" {
		port, err := strconv.Atoi(cfg.Port)
		if err != nil {
			errors = append(errors, "POSTGRES_PORT must be a valid number")
		} else if port < 1 || port > 65535 {
			errors = append(errors, "POSTGRES_PORT must be between 1 and 65535")
		}
	}

	// Validate MaxRetries is non-negative
	if cfg.MaxRetries < 0 {
		errors = append(errors, "DB_MAX_RETRIES must be non-negative")
	}

	// Validate RetryDelay is positive
	if cfg.RetryDelay <= 0 {
		errors = append(errors, "DB_RETRY_DELAY must be positive")
	}

	// Validate RetryDelay is not excessively large (optional, adjust threshold as needed)
	if cfg.RetryDelay > 10*time.Minute {
		errors = append(errors, "DB_RETRY_DELAY must not exceed 10 minutes")
	}

	// Return combined errors if any exist
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// ConnectDB establishes connection to PostgreSQL
func ConnectDB(cfg *Config) (*gorm.DB, error) {
	if cfg == nil {
		ctx := context.Background()
		loadedCfg, err := LoadConfigFromEnv(ctx)
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port,
	)

	log.Printf("Connecting to: %s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	time.Sleep(3 * time.Second)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
	}

	// Try connection with retries
	for i := 0; i < cfg.MaxRetries; i++ {
		log.Printf("[DB] Attempt %d/%d: connecting...", i+1, cfg.MaxRetries)

		gdb, err := gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			sqlDB, dbErr := gdb.DB()
			if dbErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				pingErr := sqlDB.PingContext(ctx)
				cancel()

				if pingErr == nil {
					log.Println("[DB] Connected successfully")

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

		log.Printf("[DB][WARN] %s. Retrying in %v...",
			simplifyDBError(err), cfg.RetryDelay)

		time.Sleep(cfg.RetryDelay)
	}

	return nil, fmt.Errorf("database connection failed after %d attempts", cfg.MaxRetries)
}

// simplifyDBError returns a user-friendly error message
func simplifyDBError(err error) string {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "password authentication failed"):
		return "invalid database credentials"
	case strings.Contains(msg, "connect"):
		return "cannot reach database server"
	case strings.Contains(msg, "timeout"):
		return "database connection timed out"
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

// TODO: use goose to automigrate
// Automigrate the provided models
func MigrateModels(db *gorm.DB, models ...any) error {
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}
	log.Println("Database migration completed successfully")
	return nil
}
