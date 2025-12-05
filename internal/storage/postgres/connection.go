package postgres

import (
	"context"
	"fmt"
	"log"
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

// LoadConfigFromEnv loads config from environment variables using go-envconfig
func LoadConfigFromEnv(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}
	cfg.LogLevel = logger.Info
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

	// Wait a bit for postgres to be ready
	time.Sleep(3 * time.Second)

	// Try connection with retries
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		log.Printf("Attempt %d/%d...", i, maxRetries)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err != nil {
			log.Printf("Failed to open: %v", err)
			if i < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, err
		}

		// Test connection
		sqlDB, _ := db.DB()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()

		if err != nil {
			log.Printf("Failed to ping: %v", err)
			if i < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, err
		}

		log.Println("Connected successfully!")
		return db, nil
	}

	return nil, fmt.Errorf("failed after %d attempts", maxRetries)
}
