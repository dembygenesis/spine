package di_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dembygenesis/spine/internal/config"
	"github.com/dembygenesis/spine/internal/di"
	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testfactory"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/observe"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  func(th *testsuite.Helper) *config.Config
		wantErr bool
		assert  func(th *testsuite.Helper, c *di.Container)
	}{
		{
			name:   "success-serves-a-request-with-a-real-token",
			config: func(th *testsuite.Helper) *config.Config { return th.AppConfig() },
			assert: func(th *testsuite.Helper, c *di.Container) {
				user := testfactory.New[testfactory.UserFactory](th).Subject
				tok, err := c.JWT.Issue(user.ID)
				require.NoError(th.T, err)

				req := httptest.NewRequest(http.MethodPost, org.PathOrganizations, bytes.NewBufferString(`{"name":"acme"}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+tok)
				rec := httptest.NewRecorder()
				c.Handler.ServeHTTP(rec, req)
				require.Equalf(th.T, http.StatusCreated, rec.Code, "resp: %s", rec.Body)

				var role string
				require.NoError(th.T, th.BackendAppDb().Get(&role,
					`SELECT m.role FROM organization_members m JOIN organizations o ON o.id = m.organization_id
					 WHERE o.name = 'acme' AND m.user_id = $1`, user.ID))
				require.Equal(th.T, model.RoleAdmin, role)
			},
		},
		{
			name: "error-a-short-secret-refuses-to-start",
			config: func(th *testsuite.Helper) *config.Config {
				cfg := th.AppConfig()
				cfg.JWTSecret = "short"
				return cfg
			},
			wantErr: true,
		},
		{
			name: "error-an-unreachable-database-refuses-to-start",
			config: func(th *testsuite.Helper) *config.Config {
				cfg := th.AppConfig()
				cfg.DBPort = 1
				return cfg
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			c, err := di.New(context.Background(), tt.config(th), observe.NewTestLogger())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })
			tt.assert(th, c)
		})
	}
}
