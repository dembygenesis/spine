package testsuite

import "github.com/dembygenesis/spine/internal/migration"

// SchemaHash exposes the current migrations' hash to the external tests.
func SchemaHash() string { return schemaHash(migration.Stmts) }
