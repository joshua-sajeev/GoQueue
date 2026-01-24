package app

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewApiApp(t *testing.T) {
	mockDb, _, err := sqlmock.New()
	assert.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn: mockDb,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	cfg := &config.Config{
		ServerPort: "8080",
	}

	application := NewApiApp(db, cfg)

	assert.NotNil(t, application)
	assert.Equal(t, cfg, application.Config)
	assert.Equal(t, db, application.DB)
	assert.NotNil(t, application.JobHandler)
	assert.NotNil(t, application.Router)
	assert.NotNil(t, application.Server)
}

func TestApp_Ready_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	app := &ApiApp{DB: db}

	err = app.Ready()
	assert.NoError(t, err)
}

func TestApp_Ready_DBNil(t *testing.T) {
	app := &ApiApp{DB: nil}

	err := app.Ready()
	assert.Error(t, err)
}

type mockServer struct {
	started  bool
	shutdown bool
}

func (m *mockServer) ListenAndServe() error {
	m.started = true
	return nil
}

func (m *mockServer) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return nil
}

func TestApp_Run_ShutdownFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockSrv := &mockServer{}
	app := &ApiApp{
		Server: mockSrv,
		DB:     nil, // DB close is guarded
		Config: &config.Config{
			ServerPort: "8080",
		},
	}

	go app.Run(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	assert.True(t, mockSrv.started)
	assert.True(t, mockSrv.shutdown)
}
