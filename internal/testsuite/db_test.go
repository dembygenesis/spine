package testsuite_test

import (
	"testing"

	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/stretchr/testify/require"
)

func TestUseDB(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "success-organizations-table-exists",
			query: `SELECT to_regclass('organizations') IS NOT NULL`,
		},
		{
			name:  "success-migrations-are-recorded",
			query: `SELECT version = 1 AND NOT dirty FROM schema_migrations`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testsuite.UseDB(t)

			var ok bool
			require.NoError(t, db.Get(&ok, tt.query))
			require.True(t, ok)
		})
	}
}
