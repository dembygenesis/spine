package config_test

import (
	"testing"

	"github.com/dembygenesis/spine/internal/config"
	"github.com/stretchr/testify/require"
)

func full() map[string]string {
	return map[string]string{
		config.EnvDBHost:    "localhost",
		config.EnvDBPort:    "5432",
		config.EnvDBUser:    "spine",
		config.EnvDBPass:    "spine",
		config.EnvDBName:    "spine",
		config.EnvJWTSecret: "a-secret-that-is-at-least-32-bytes",
		config.EnvPort:      "8080",
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     func() map[string]string
		wantErr string
	}{
		{
			name: "success-everything-set",
			env:  full,
		},
		{
			name: "error-names-every-missing-variable-at-once",
			env: func() map[string]string {
				e := full()
				delete(e, config.EnvDBHost)
				delete(e, config.EnvJWTSecret)
				return e
			},
			wantErr: "missing environment: SPINE_DB_HOST, SPINE_JWT_SECRET",
		},
		{
			name: "error-a-port-that-is-not-a-number",
			env: func() map[string]string {
				e := full()
				e[config.EnvPort] = "eighty"
				return e
			},
			wantErr: `SPINE_PORT="eighty" is not a port`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := tt.env()
			_, err := config.Load(func(k string) string { return env[k] })
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
