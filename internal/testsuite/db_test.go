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
		assert func(t *testing.T)
	}{
		{
			name: "success-schema-is-migrated",
			assert: func(t *testing.T) {
				db := testsuite.UseDB(t)

				var ok bool
				require.NoError(t, db.Get(&ok, `SELECT version = 1 AND NOT dirty FROM schema_migrations`))
				require.True(t, ok)
			},
		},
		{
			name: "success-parallel-tests-see-only-their-own-rows",
			assert: func(t *testing.T) {
				db := testsuite.UseDB(t)
				insertAcme(t, db)
				require.Equal(t, 1, countOrganizations(t, db))
			},
		},
		{
			name: "success-the-same-row-again-in-another-test",
			assert: func(t *testing.T) {
				db := testsuite.UseDB(t)
				insertAcme(t, db)
				require.Equal(t, 1, countOrganizations(t, db))
			},
		},
		{
			name: "success-database-is-dropped-after-the-test",
			assert: func(t *testing.T) {
				var name string
				t.Run("inner", func(t *testing.T) {
					db := testsuite.UseDB(t)
					require.NoError(t, db.Get(&name, `SELECT current_database()`))
				})
				require.False(t, databaseExists(t, name), "database %s outlived its test", name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t)
		})
	}
}

func insertAcme(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO organizations (name) VALUES ('acme')`)
	require.NoError(t, err)
}

func countOrganizations(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.Get(&n, `SELECT count(*) FROM organizations`))
	return n
}

func databaseExists(t *testing.T, name string) bool {
	t.Helper()
	c, err := pg.GetContainer()
	require.NoError(t, err)
	admin, err := sqlx.Connect("postgres", c.Config.DSN(c.Config.Database))
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()

	var exists bool
	require.NoError(t, admin.Get(&exists, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name))
	return exists
}
