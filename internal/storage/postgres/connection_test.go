package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]string
		want    *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			envs: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpassword",
				"POSTGRES_DB":       "testdb",
				"POSTGRES_HOST":     "testhost",
				"POSTGRES_PORT":     "5432",
				"DB_MAX_RETRIES":    "5",
				"DB_RETRY_DELAY":    "2s",
				"DB_LOG_LEVEL":      "silent",
			},
			want: &Config{
				User:           "testuser",
				Password:       "testpassword",
				Database:       "testdb",
				Host:           "testhost",
				Port:           "5432",
				MaxRetries:     5,
				RetryDelay:     2 * time.Second,
				LogLevelString: "silent",
				LogLevel:       1,
			},
			wantErr: false,
		},
		{
			name:    "envconfig_process_error",
			envs:    map[string]string{},
			want:    nil,
			wantErr: true,
			errMsg:  "failed to process env config",
		},
		{
			name: "missing_password",
			envs: map[string]string{
				"POSTGRES_USER": "testuser",
				"POSTGRES_DB":   "testdb",
				"POSTGRES_HOST": "testhost",
				"POSTGRES_PORT": "1234",
			},
			wantErr: true,
			errMsg:  "Password: missing required value: POSTGRES_PASSWORD",
		},
		{
			name: "database_with_space",
			envs: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpassword",
				"POSTGRES_DB":       "  ",
				"POSTGRES_HOST":     "testhost",
				"POSTGRES_PORT":     "5432",
				"DB_MAX_RETRIES":    "5",
				"DB_RETRY_DELAY":    "2s",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "POSTGRES_DB is required",
		},
		{
			name: "invalid_host_with_spaces",
			envs: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpassword",
				"POSTGRES_DB":       "testdb",
				"POSTGRES_HOST":     "  ",
				"POSTGRES_PORT":     "5432",
				"DB_MAX_RETRIES":    "5",
				"DB_RETRY_DELAY":    "2s",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "POSTGRES_HOST is required",
		},
		{
			name: "invalid_port_format",
			envs: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpassword",
				"POSTGRES_DB":       "testdb",
				"POSTGRES_HOST":     "testhost",
				"POSTGRES_PORT":     "invalid",
				"DB_MAX_RETRIES":    "5",
				"DB_RETRY_DELAY":    "2s",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "POSTGRES_PORT must be a valid number",
		},
		{
			name: "special_characters_in_password",
			envs: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "p@ssw0rd!#$%^&*()",
				"POSTGRES_DB":       "testdb",
				"POSTGRES_HOST":     "testhost",
				"POSTGRES_PORT":     "5432",
				"DB_MAX_RETRIES":    "3",
				"DB_RETRY_DELAY":    "1s",
			},
			want: &Config{
				User:           "testuser",
				Password:       "p@ssw0rd!#$%^&*()",
				Database:       "testdb",
				Host:           "testhost",
				Port:           "5432",
				MaxRetries:     3,
				RetryDelay:     1 * time.Second,
				LogLevelString: "warn",
				LogLevel:       3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "envconfig_process_error" {
				originalEnvProcess := envProcess
				envProcess = func(ctx context.Context, i any, mus ...envconfig.Mutator) error {
					return fmt.Errorf("mock envconfig error")
				}
				defer func() {
					envProcess = originalEnvProcess
				}()
			}

			for key, val := range tt.envs {
				os.Setenv(key, val)
				t.Cleanup(func() { os.Unsetenv(key) })
			}

			got, err := LoadConfigFromEnv(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.User, got.User)
			assert.Equal(t, tt.want.Password, got.Password)
			assert.Equal(t, tt.want.Database, got.Database)
			assert.Equal(t, tt.want.Host, got.Host)
			assert.Equal(t, tt.want.Port, got.Port)
			assert.Equal(t, tt.want.MaxRetries, got.MaxRetries)
			assert.Equal(t, tt.want.RetryDelay, got.RetryDelay)
			assert.Equal(t, tt.want.LogLevelString, got.LogLevelString)
			assert.Equal(t, tt.want.LogLevel, got.LogLevel)
		})
	}
}
