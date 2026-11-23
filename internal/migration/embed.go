// Package migration embeds the schema, so the binary and the tests carry the
// exact same SQL with no files to ship alongside them.
package migration

import "embed"

// Stmts holds every NNNNNN_name.up.sql / .down.sql pair, in golang-migrate's
// naming. The number orders them; a duplicate number refuses to load.
//
//go:embed *.sql
var Stmts embed.FS
