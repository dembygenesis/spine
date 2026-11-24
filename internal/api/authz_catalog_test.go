package api_test

import (
	"context"
	"testing"

	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/auth"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/stretchr/testify/require"
)

// A gated pair with no role_permissions row compiles, starts and serves: every
// caller just gets a 403. This test is what turns that silent runtime refusal
// into a red build.
func TestGatedPermissionsExistInTheCatalog(t *testing.T) {
	tests := []struct {
		name     string
		register func(th *testsuite.Helper) []string
	}{
		{
			name: "success-every-org-route-pair-has-a-row",
			register: func(th *testsuite.Helper) []string {
				mux := th.SetupAPI()
				tr, err := pg.NewTransactor(th.BackendAppDb())
				require.NoError(th.T, err)
				require.NoError(th.T, org.Register(mux, tr))

				catalog, err := auth.LoadCatalog(context.Background(), th.BackendAppDb())
				require.NoError(th.T, err)

				var missing []string
				for _, p := range mux.GatedPermissions() {
					if !catalog.Has(p.Resource, p.Action) {
						missing = append(missing, p.Resource+":"+p.Action)
					}
				}
				return missing
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())
			require.Empty(t, tt.register(th), "gated with no role_permissions row: add a migration")
		})
	}
}
