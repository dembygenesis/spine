package testsuite

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"

	"github.com/dembygenesis/spine/internal/migration"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/jmoiron/sqlx"
)

// Migrating a database per test costs every migration, every test. Postgres can
// instead copy a database: CREATE DATABASE x TEMPLATE y copies y's files and
// replays no SQL. So the migrations run once, into a template no test ever
// writes to, and each test clones it.
//
// Three things make that safe when many tests and many test binaries do it at
// once:
//
//	the NAME carries a hash of the migrations, so a changed migration names a
//	  new template and a stale one is never cloned
//	an ADVISORY LOCK guards creation, because sync.Once only covers one
//	  process and go test runs one process per package
//	the template is built under a SCRATCH name and renamed when complete, so a
//	  crash mid-migration never leaves a template that looks finished
const templatePrefix = "spine_tmpl_"

// ensureTemplate returns the name of a migrated template, building it on first
// use. Any number of processes may call it at once: the losers wait on the lock,
// then find the template already there.
func ensureTemplate(admin *sqlx.DB, cfg *pg.ContainerConfig) (string, error) {
	name := templatePrefix + schemaHash(migration.Stmts)

	// Session-scoped, so it is released if this process dies holding it.
	if _, err := admin.Exec(`SELECT pg_advisory_lock($1)`, lockKey(name)); err != nil {
		return "", fmt.Errorf("lock template: %w", err)
	}
	defer func() { _, _ = admin.Exec(`SELECT pg_advisory_unlock($1)`, lockKey(name)) }()

	var exists bool
	if err := admin.Get(&exists, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name); err != nil {
		return "", fmt.Errorf("check template: %w", err)
	}
	if exists {
		return name, nil
	}

	scratch := name + "_building"
	if _, err := admin.Exec(`DROP DATABASE IF EXISTS "` + scratch + `"`); err != nil {
		return "", fmt.Errorf("drop stale scratch: %w", err)
	}
	if _, err := admin.Exec(`CREATE DATABASE "` + scratch + `"`); err != nil {
		return "", fmt.Errorf("create scratch: %w", err)
	}
	if err := migrateScratch(cfg, scratch); err != nil {
		return "", err
	}

	// Postgres will not rename or copy a database with a session attached, and
	// golang-migrate's own connection outlives the one closed above.
	if _, err := admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, scratch); err != nil {
		return "", fmt.Errorf("detach scratch: %w", err)
	}
	if _, err := admin.Exec(`ALTER DATABASE "` + scratch + `" RENAME TO "` + name + `"`); err != nil {
		return "", fmt.Errorf("promote template: %w", err)
	}
	return name, nil
}

// migrateScratch migrates the scratch database and closes its connection before
// returning: a leaked session here surfaces later as a clone that fails to start.
func migrateScratch(cfg *pg.ContainerConfig, scratch string) error {
	db, err := sqlx.Connect("postgres", cfg.DSN(scratch))
	if err != nil {
		return fmt.Errorf("connect scratch: %w", err)
	}
	if err := runMigrations(db, scratch); err != nil {
		_ = db.Close()
		return fmt.Errorf("migrate scratch: %w", err)
	}
	return db.Close()
}

// schemaHash fingerprints every file in migrations: the same files always give
// the same hash, and any change to any of them gives a different one.
//
// The names are sorted because fs.WalkDir's order is not a promise, and a hash
// that depends on iteration order rebuilds the template for nothing.
func schemaHash(migrations fs.FS) string {
	var names []string
	_ = fs.WalkDir(migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	sort.Strings(names)

	sum := sha256.New()
	for _, n := range names {
		b, err := fs.ReadFile(migrations, n)
		if err != nil {
			continue
		}
		sum.Write([]byte(n))
		sum.Write(b)
	}
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// lockKey turns a template name into the int64 an advisory lock takes.
func lockKey(name string) int64 {
	sum := sha256.Sum256([]byte(name))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
