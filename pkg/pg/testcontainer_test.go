package pg_test

import (
	"strings"
	"testing"

	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestGetContainer(t *testing.T) {
	tests := []struct {
		name   string
		assert func(t *testing.T, c *pg.TestContainer)
	}{
		{
			name: "success-answers-sql",
			assert: func(t *testing.T, c *pg.TestContainer) {
				db, err := sqlx.Connect("postgres", c.Config.DSN(c.Config.Database))
				require.NoError(t, err)
				t.Cleanup(func() { _ = db.Close() })

				var version string
				require.NoError(t, db.Get(&version, `SELECT version()`))
				require.True(t, strings.HasPrefix(version, "PostgreSQL 15"), version)
			},
		},
		{
			name: "success-second-call-returns-the-same-container",
			assert: func(t *testing.T, c *pg.TestContainer) {
				again, err := pg.GetContainer()
				require.NoError(t, err)
				require.Same(t, c, again)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := pg.GetContainer()
			require.NoError(t, err)
			tt.assert(t, c)
		})
	}
}
