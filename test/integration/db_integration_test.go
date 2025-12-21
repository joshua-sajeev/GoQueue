package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB   *sql.DB
	testDSN  string
	testPort string
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	pool.MaxWait = 60 * time.Second

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	pg, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17-alpine",
		Env: []string{
			"POSTGRES_USER=testuser",
			"POSTGRES_PASSWORD=testpass",
			"POSTGRES_DB=example",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start postgres container: %s", err)
	}

	testPort = pg.GetPort("5432/tcp")
	testDSN = fmt.Sprintf(
		"host=localhost user=testuser password=testpass dbname=example port=%s sslmode=disable TimeZone=UTC",
		testPort,
	)

	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("postgres", testDSN)
		if err != nil {
			log.Printf("Failed to open database: %v", err)
			return err
		}

		testDB.SetMaxOpenConns(10)
		testDB.SetMaxIdleConns(5)
		testDB.SetConnMaxLifetime(5 * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := testDB.PingContext(ctx); err != nil {
			testDB.Close()
			return err
		}

		var version string
		err = testDB.QueryRowContext(ctx, "SELECT version()").Scan(&version)
		if err != nil {
			testDB.Close()
			return err
		}

		// Run Goose migrations
		if err := runMigrations(testDB); err != nil {
			log.Printf("Failed to run migrations: %v", err)
			testDB.Close()
			return err
		}

		// Create additional test database for Test 2
		_, err = testDB.Exec("CREATE DATABASE example2")
		if err != nil {
			log.Printf("Warning: Could not create example2 database: %v", err)
		}

		return nil
	}); err != nil {
		log.Fatalf("Could not connect to postgres: %s", err)
	}

	os.Setenv("POSTGRES_USER", "testuser")
	os.Setenv("POSTGRES_PASSWORD", "testpass")
	os.Setenv("POSTGRES_DB", "example")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", testPort)
	os.Setenv("DB_MAX_RETRIES", "3")
	os.Setenv("DB_RETRY_DELAY", "100ms")
	os.Setenv("DB_LOG_LEVEL", "1")

	code := m.Run()

	if testDB != nil {
		testDB.Close()
	}

	if err := pool.Purge(pg); err != nil {
		log.Fatalf("Could not purge postgres container: %s", err)
	}

	os.Exit(code)
}

func runMigrations(db *sql.DB) error {
	// Get the path to migrations directory
	// Adjust based on where this file is and where migrations are stored
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "../..")
	migrationsDir := filepath.Join(projectRoot, "migrations")

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", migrationsDir)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run goose migrations: %w", err)
	}

	return nil
}

func TestConnectDB(t *testing.T) {
	tests := []struct {
		name        string
		config      *postgres.Config
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		errContains string
		validate    func(t *testing.T, db *gorm.DB)
	}{
		{
			name:    "Test 1: Verify TestMain setup is correct",
			config:  nil,
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB) {
				require.NotNil(t, db)

				assert.Equal(t, "testuser", os.Getenv("POSTGRES_USER"))
				assert.Equal(t, "testpass", os.Getenv("POSTGRES_PASSWORD"))
				assert.Equal(t, "example", os.Getenv("POSTGRES_DB"))
				assert.Equal(t, "localhost", os.Getenv("POSTGRES_HOST"))
				assert.Equal(t, testPort, os.Getenv("POSTGRES_PORT"))
				assert.NotEmpty(t, testPort, "testPort should be set by TestMain")

				sqlDB, err := db.DB()
				require.NoError(t, err)
				assert.NoError(t, sqlDB.Ping())

				var result int
				err = db.Raw("SELECT 1").Scan(&result).Error
				require.NoError(t, err)
				assert.Equal(t, 1, result)

				var dbName string
				err = db.Raw("SELECT current_database()").Scan(&dbName).Error
				require.NoError(t, err)
				assert.Equal(t, "example", dbName)

				stats := sqlDB.Stats()
				assert.Equal(t, 50, stats.MaxOpenConnections)
				assert.GreaterOrEqual(t, stats.Idle, 0)
			},
		},
		{
			name:    "Test 2: Load from environment with different database",
			config:  nil,
			wantErr: false,
			setupEnv: func() {
				os.Setenv("POSTGRES_DB", "example2")
			},
			cleanupEnv: func() {
				os.Setenv("POSTGRES_DB", "example")
			},
			validate: func(t *testing.T, db *gorm.DB) {
				require.NotNil(t, db)

				sqlDB, err := db.DB()
				require.NoError(t, err)
				assert.NoError(t, sqlDB.Ping())

				var dbName string
				err = db.Raw("SELECT current_database()").Scan(&dbName).Error
				require.NoError(t, err)
				assert.Equal(t, "example2", dbName, "Should be connected to example2 database")

				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS test_table (
						id SERIAL PRIMARY KEY,
						name VARCHAR(100)
					)
				`).Error
				require.NoError(t, err, "Should be able to create table in example2")

				var tableExists bool
				err = db.Raw(`
					SELECT EXISTS (
						SELECT FROM information_schema.tables 
						WHERE table_schema = 'public' 
						AND table_name = 'test_table'
					)
				`).Scan(&tableExists).Error
				require.NoError(t, err)
				assert.True(t, tableExists, "test_table should exist in example2")

				db.Exec("DROP TABLE IF EXISTS test_table")

				stats := sqlDB.Stats()
				assert.Equal(t, 50, stats.MaxOpenConnections)
			},
		},
		{
			name: "Test 3: Successful connection with explicit config",
			config: &postgres.Config{
				User:       "testuser",
				Password:   "testpass",
				Host:       "localhost",
				Port:       testPort,
				Database:   "example",
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				LogLevel:   logger.Silent,
			},
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB) {
				require.NotNil(t, db)

				sqlDB, err := db.DB()
				require.NoError(t, err)
				assert.NoError(t, sqlDB.Ping())

				stats := sqlDB.Stats()
				assert.Equal(t, 50, stats.MaxOpenConnections)

				var dbName string
				err = db.Raw("SELECT current_database()").Scan(&dbName).Error
				require.NoError(t, err)
				assert.Equal(t, "example", dbName)

				tx := db.Begin()
				require.NotNil(t, tx)
				assert.NoError(t, tx.Error)
				assert.NoError(t, tx.Rollback().Error)
			},
		},
		{
			name:   "Test 4: Failed connection with missing environment variables",
			config: nil,
			setupEnv: func() {
				os.Unsetenv("POSTGRES_USER")
				os.Unsetenv("POSTGRES_PASSWORD")
				os.Unsetenv("POSTGRES_HOST")
				os.Unsetenv("POSTGRES_PORT")
				os.Unsetenv("POSTGRES_DB")
			},
			cleanupEnv: func() {
				os.Setenv("POSTGRES_USER", "testuser")
				os.Setenv("POSTGRES_PASSWORD", "testpass")
				os.Setenv("POSTGRES_HOST", "localhost")
				os.Setenv("POSTGRES_PORT", testPort)
				os.Setenv("POSTGRES_DB", "example")
			},
			wantErr:     true,
			errContains: "",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
		{
			name: "Test 5a: Connection refused (wrong port)",
			config: &postgres.Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           "19999",
				Database:       "example",
				MaxRetries:     2,
				RetryDelay:     5 * time.Millisecond,
				LogLevel:       logger.Silent,
				ConnectTimeout: 1,
			},
			wantErr:     true,
			errContains: "database connection failed after 2 attempts",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
		{
			name: "Test 5b: Invalid credentials",
			config: &postgres.Config{
				User:           "testuser",
				Password:       "wrongpass",
				Host:           "localhost",
				Port:           testPort,
				Database:       "example",
				MaxRetries:     2,
				RetryDelay:     5 * time.Millisecond,
				LogLevel:       logger.Silent,
				ConnectTimeout: 1,
			},
			wantErr:     true,
			errContains: "database connection failed after 2 attempts",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
		{
			name: "Test 5c: Non-existent database",
			config: &postgres.Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "localhost",
				Port:           testPort,
				Database:       "nonexistent_db",
				MaxRetries:     4,
				RetryDelay:     5 * time.Millisecond,
				LogLevel:       logger.Silent,
				ConnectTimeout: 1,
			},
			wantErr:     true,
			errContains: "database connection failed after 4 attempts",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
		{
			name: "Test 5d: Network timeout (non-routable IP)",
			config: &postgres.Config{
				User:           "testuser",
				Password:       "testpass",
				Host:           "192.0.2.1",
				Port:           "5432",
				Database:       "example",
				MaxRetries:     2,
				RetryDelay:     5 * time.Millisecond,
				LogLevel:       logger.Silent,
				ConnectTimeout: 1,
			},
			wantErr:     true,
			errContains: "database connection failed after 2 attempts",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
		{
			name: "Test 6: MaxRetries = 0 should fail immediately",
			config: &postgres.Config{
				User:       "testuser",
				Password:   "testpass",
				Host:       "invalid-host",
				Port:       testPort,
				Database:   "example",
				MaxRetries: 0,
				RetryDelay: 100 * time.Millisecond,
				LogLevel:   logger.Silent,
			},
			wantErr:     true,
			errContains: "database connection failed after 0 attempts",
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},

		{
			name: "Test 7: Missing fields in explicit config",
			config: &postgres.Config{
				User:       "testuser",
				Host:       "localhost",
				Port:       testPort,
				Database:   "example",
				MaxRetries: 1,
				RetryDelay: 50 * time.Millisecond,
				LogLevel:   logger.Silent,
			},
			wantErr: true,
			validate: func(t *testing.T, db *gorm.DB) {
				assert.Nil(t, db)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with timeout for each test
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			db, err := postgres.ConnectDB(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err, "Expected an error but got none")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, db, "Expected db to be nil on error")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
				require.NotNil(t, db, "Expected db to be non-nil")
				if tt.validate != nil {
					tt.validate(t, db)
				}
				sqlDB, err := db.DB()
				if err == nil {
					sqlDB.Close()
				}
			}

			if tt.cleanupEnv != nil {
				tt.cleanupEnv()
			}
		})
	}
}

// setupTestDB returns a fresh DB connection and context with automatic cleanup
// Each test gets its own connection to avoid connection pool issues
func setupTestDB(tb testing.TB) (*gorm.DB, context.Context) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	tb.Cleanup(cancel)

	testConfig := &postgres.Config{
		User:       "testuser",
		Password:   "testpass",
		Host:       "localhost",
		Port:       testPort,
		Database:   "example",
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		LogLevel:   logger.Silent,
	}

	db, err := postgres.ConnectDB(ctx, testConfig)
	require.NoError(tb, err)

	// Clean up the jobs table at the start of each test / benchmark
	if err := db.Exec("DELETE FROM jobs").Error; err != nil {
		tb.Logf("Warning: Failed to clean jobs table: %v", err)
	}

	// Register cleanup
	tb.Cleanup(func() {
		closeTestDB(db)
	})

	return db, ctx
}

// closeTestDB closes a DB connection
func closeTestDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	if sqlDB != nil {
		sqlDB.Close()
	}
}
