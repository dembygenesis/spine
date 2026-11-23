// Package testsuite is the harness every database test runs on.
package testsuite

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/dembygenesis/spine/internal/migration"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// UseDB gives the calling test a database of its own, cloned from the migrated
// template and dropped when the test ends. No two tests ever see each other's
// rows, and no test pays for the migrations.
func UseDB(t *testing.T) *sqlx.DB {
	t.Helper()
	started := time.Now()

	c, err := pg.GetContainer()
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

	t.Cleanup(func() {
		_ = db.Close()
		// Postgres refuses to drop a database with sessions attached, and a test
		// may have left one open.
		_, _ = admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, name)
		_, _ = admin.Exec(`DROP DATABASE "` + name + `"`)
		_ = admin.Close()
	})
	return db
}

// runMigrations applies every pending up migration. Already-applied ones are
// skipped, which golang-migrate reports as ErrNoChange.
func runMigrations(db *sqlx.DB, dbName string) error {
	src, err := iofs.New(migration.Stmts, ".")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{DatabaseName: dbName})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, dbName, driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
