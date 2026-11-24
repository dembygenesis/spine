// Package testsuite is the harness every database test runs on.
package testsuite

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// UseBackendDB gives the test a database of its own, cloned from the migrated
// template, and returns the func that drops it:
//
//	t.Cleanup(th.UseBackendDB())
//
// It returns the cleanup rather than registering it so the call site shows the
// database's lifetime, and a test can drop it early to check what dropping does.
func (h *Helper) UseBackendDB() func() {
	t := h.T
	t.Helper()
	started := time.Now()

	// Which container this test lands on is the pool's decision: see pool.go.
	c, err := dbPool.next()
	if err != nil {
		t.Fatalf("test container: %v", err)
	}

	// The admin connection sits on the container's default database, because
	// Postgres cannot create or drop the database a session is connected to.
	admin, err := sqlx.Connect("postgres", c.Config.DSN(c.Config.Database))
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}

	template, err := ensureTemplate(admin, c.Config)
	if err != nil {
		_ = admin.Close()
		t.Fatalf("template: %v", err)
	}

	// Postgres identifiers cannot contain hyphens.
	name := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `" TEMPLATE "` + template + `"`); err != nil {
		_ = admin.Close()
		t.Fatalf("create %s: %v", name, err)
	}

	db, err := sqlx.Connect("postgres", c.Config.DSN(name))
	if err != nil {
		_ = admin.Close()
		t.Fatalf("connect %s: %v", name, err)
	}
	log.Printf("[testsuite] provisioned %s in %s", name, time.Since(started).Round(time.Millisecond))

	h.db = db
	h.dbHost, h.dbPort, h.dbName = c.Config.Host, c.Config.Port, name
	return func() {
		_ = db.Close()
		// Postgres refuses to drop a database with sessions attached, and a test
		// may have left one open.
		_, _ = admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, name)
		_, _ = admin.Exec(`DROP DATABASE "` + name + `"`)
		_ = admin.Close()
	}
}
