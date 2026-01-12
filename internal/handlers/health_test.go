package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHealthCheckHandler_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(mock sqlmock.Sqlmock)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - Database is up",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing() // GORM Init Ping
				mock.ExpectPing() // Handler's PingContext
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name: "Failure - Database unreachable",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
				mock.ExpectPing().WillReturnError(errors.New("connection lost"))
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "database is unavailable",
		},
		{
			name: "Failure - Database Timeout",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()

				mock.ExpectPing().
					WillDelayFor(3 * time.Second).
					WillReturnError(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "database is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
			defer mockDB.Close()

			tt.setupMock(mock)

			dialer := postgres.New(postgres.Config{Conn: mockDB})
			db, _ := gorm.Open(dialer, &gorm.Config{})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/health", nil)

			handler := HealthCheckHandler(db)
			handler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
