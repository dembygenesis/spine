package testsuite_test

import (
	"testing"

	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUseDB(t *testing.T) {
	tests := []struct {
		name   string
		assert func(th *testsuite.Helper)
	}{
		{
			name: "success-schema-is-migrated",
			assert: func(th *testsuite.Helper) {
				th.T.Cleanup(th.UseBackendDB())

				var ok bool
				require.NoError(th.T, th.BackendAppDb().Get(&ok, `SELECT version = 1 AND NOT dirty FROM schema_migrations`))
				require.True(th.T, ok)
			},
		},
		{
			name: "success-parallel-tests-see-only-their-own-rows",
			assert: func(th *testsuite.Helper) {
				th.T.Cleanup(th.UseBackendDB())
				insertAcme(th)
				require.Equal(th.T, 1, countOrganizations(th))
			},
		},
		{
			name: "success-the-same-row-again-in-another-test",
			assert: func(th *testsuite.Helper) {
				th.T.Cleanup(th.UseBackendDB())
				insertAcme(th)
				require.Equal(th.T, 1, countOrganizations(th))
			},
		},
		{
			name: "success-database-is-dropped-after-the-test",
			assert: func(th *testsuite.Helper) {
				var name string
				th.T.Run("inner", func(t *testing.T) {
					inner := testsuite.New(t)
					t.Cleanup(inner.UseBackendDB())
					require.NoError(t, inner.BackendAppDb().Get(&name, `SELECT current_database()`))
				})
				require.False(th.T, databaseExists(th.T, name), "database %s outlived its test", name)
			},
		},
		{
			name: "success-one-finished-template-no-scratch-left",
			assert: func(th *testsuite.Helper) {
				th.T.Cleanup(th.UseBackendDB())
				require.Equal(th.T, []string{"done"}, templateStates(th.T))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(testsuite.New(t))
		})
	}
}

func insertAcme(th *testsuite.Helper) {
	th.T.Helper()
	_, err := th.BackendAppDb().Exec(`INSERT INTO organizations (name) VALUES ('acme')`)
	require.NoError(th.T, err)
}

func countOrganizations(th *testsuite.Helper) int {
	th.T.Helper()
	var n int
	require.NoError(th.T, th.BackendAppDb().Get(&n, `SELECT count(*) FROM organizations`))
	return n
}

// templateStates lists every spine template on the container as "done" or
// "building". Other runs with other migrations may leave their own templates,
// so only the ones for the current migrations count.
func templateStates(t *testing.T) []string {
	t.Helper()
	admin := adminDB(t)

	var states []string
	require.NoError(t, admin.Select(&states, `
		SELECT CASE WHEN datname LIKE '%\_building' THEN 'building' ELSE 'done' END
		FROM pg_database
		WHERE datname LIKE 'spine\_tmpl\_%' AND datname LIKE '%' || $1 || '%'`,
		testsuite.SchemaHash()))
	return states
}

func adminDB(t *testing.T) *sqlx.DB {
	t.Helper()
	c, err := pg.GetContainer()
	require.NoError(t, err)
	admin, err := sqlx.Connect("postgres", c.Config.DSN(c.Config.Database))
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	return admin
}

func databaseExists(t *testing.T, name string) bool {
	t.Helper()
	var exists bool
	require.NoError(t, adminDB(t).Get(&exists, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name))
	return exists
}
