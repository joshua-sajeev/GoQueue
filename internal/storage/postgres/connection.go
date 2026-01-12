package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDB establishes connection to PostgreSQL with context support
func ConnectDB(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil {
		loadedCfg, err := config.LoadConfigFromEnv(ctx)
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
