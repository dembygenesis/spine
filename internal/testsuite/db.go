// Package testsuite is the harness every database test runs on.
package testsuite

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dembygenesis/spine/internal/migration"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// UseDB connects to the container's database and brings its schema up to date.
// Every test that calls it shares that one database.
func UseDB(t *testing.T) *sqlx.DB {
	t.Helper()

	c, err := pg.GetContainer()
	if err != nil {
		t.Fatalf("test container: %v", err)
	}

	db, err := sqlx.Connect("postgres", c.Config.DSN(c.Config.Database))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := runMigrations(db, c.Config.Database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
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
