package app

import (
	"context"
	"errors"
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

func TestApp_Ready(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *ApiApp
		wantErr bool
	}{
		{
			name: "success with valid db",
			setup: func() *ApiApp {
				db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				return &ApiApp{DB: db}
			},
			wantErr: false,
		},
		{
			name: "error when db is nil",
			setup: func() *ApiApp {
				return &ApiApp{DB: nil}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setup()
			err := app.Ready()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApp_Run(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() Server
		setupDB     func() (*gorm.DB, sqlmock.Sqlmock)
		validate    func(*testing.T, Server, sqlmock.Sqlmock)
		waitTime    time.Duration
	}{
		{
			name: "normal shutdown flow",
			setupServer: func() Server {
				return &mockServer{}
			},
			setupDB: func() (*gorm.DB, sqlmock.Sqlmock) {
				return nil, nil
			},
			validate: func(t *testing.T, srv Server, _ sqlmock.Sqlmock) {
				ms := srv.(*mockServer)
				assert.True(t, ms.started)
				assert.True(t, ms.shutdown)
			},
			waitTime: 50 * time.Millisecond,
		},
		{
			name: "shutdown error is logged",
			setupServer: func() Server {
				return &mockServerWithShutdownError{
					shutdownErr: errors.New("shutdown failed"),
				}
			},
			setupDB: func() (*gorm.DB, sqlmock.Sqlmock) {
				return nil, nil
			},
			validate: func(t *testing.T, srv Server, _ sqlmock.Sqlmock) {
				ms := srv.(*mockServerWithShutdownError)
				assert.True(t, ms.shutdownCalled)
			},
			waitTime: 100 * time.Millisecond,
		},
		{
			name: "db close error is logged",
			setupServer: func() Server {
				return &mockServer{}
			},
			setupDB: func() (*gorm.DB, sqlmock.Sqlmock) {
				mockDb, mock, _ := sqlmock.New()
				mock.ExpectClose().WillReturnError(errors.New("close failed"))
				dialector := postgres.New(postgres.Config{Conn: mockDb})
				db, _ := gorm.Open(dialector, &gorm.Config{})
				return db, mock
			},
			validate: func(t *testing.T, srv Server, mock sqlmock.Sqlmock) {
				ms := srv.(*mockServer)
				assert.True(t, ms.shutdown)
				if mock != nil {
					assert.NoError(t, mock.ExpectationsWereMet())
				}
			},
			waitTime: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			srv := tt.setupServer()
			db, mock := tt.setupDB()

			app := &ApiApp{
				Server: srv,
				DB:     db,
				Config: &config.Config{
					ServerPort: "8080",
				},
			}

			go app.Run(ctx)
			time.Sleep(50 * time.Millisecond)
			cancel()
			time.Sleep(tt.waitTime)

			tt.validate(t, srv, mock)
		})
	}
}
