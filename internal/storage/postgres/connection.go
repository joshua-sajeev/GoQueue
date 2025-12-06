package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	User       string        `env:"POSTGRES_USER,default=postgres"`
	Password   string        `env:"POSTGRES_PASSWORD,default=postgres"`
	Host       string        `env:"POSTGRES_HOST,default=database"`
	Port       string        `env:"POSTGRES_PORT,default=5432"`
	Database   string        `env:"POSTGRES_DB,default=taskdb"`
	MaxRetries int           `env:"DB_MAX_RETRIES,default=10"`
	RetryDelay time.Duration `env:"DB_RETRY_DELAY,default=2s"`
	LogLevel   logger.LogLevel
}

func LoadConfigFromEnv(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}
	//TODO: move this to a env file
	cfg.LogLevel = logger.Silent
	return &cfg, nil
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
