package testsuite

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestSchemaHash(t *testing.T) {
	base := fstest.MapFS{
		"000001_a.up.sql": {Data: []byte("CREATE TABLE a ();")},
		"000002_b.up.sql": {Data: []byte("CREATE TABLE b ();")},
	}

	tests := []struct {
		name      string
		other     fstest.MapFS
		wantEqual bool
	}{
		{
			name: "success-same-files-same-hash",
			other: fstest.MapFS{
				"000001_a.up.sql": {Data: []byte("CREATE TABLE a ();")},
				"000002_b.up.sql": {Data: []byte("CREATE TABLE b ();")},
			},
			wantEqual: true,
		},
		{
			name: "success-an-edited-migration-changes-the-hash",
			other: fstest.MapFS{
				"000001_a.up.sql": {Data: []byte("CREATE TABLE a (id INT);")},
				"000002_b.up.sql": {Data: []byte("CREATE TABLE b ();")},
			},
			wantEqual: false,
		},
		{
			name: "success-a-new-migration-changes-the-hash",
			other: fstest.MapFS{
				"000001_a.up.sql": {Data: []byte("CREATE TABLE a ();")},
				"000002_b.up.sql": {Data: []byte("CREATE TABLE b ();")},
				"000003_c.up.sql": {Data: []byte("CREATE TABLE c ();")},
			},
			wantEqual: false,
		},
		{
			name: "success-a-renamed-migration-changes-the-hash",
			other: fstest.MapFS{
				"000001_a.up.sql":       {Data: []byte("CREATE TABLE a ();")},
				"000002_renamed.up.sql": {Data: []byte("CREATE TABLE b ();")},
			},
			wantEqual: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := schemaHash(base) == schemaHash(tt.other)
			require.Equal(t, tt.wantEqual, got)
		})
	}
}
