package testsuite

import (
	"testing"

	"github.com/dembygenesis/spine/internal/config"

	"github.com/jmoiron/sqlx"
)

// Helper is the one value a test carries around: its *testing.T and, once
// provisioned, its database. Anything that needs both (fixtures, factories,
// assertions) takes a *Helper instead of two arguments that must stay paired.
type Helper struct {
	T      *testing.T
	Fakers *Fakers

	db     *sqlx.DB
	dbHost string
	dbPort int
	dbName string
}

// New starts a Helper for one test or subtest.
func New(t *testing.T) *Helper {
	t.Helper()
	return &Helper{T: t, Fakers: newFakers()}
}

// BackendAppDb is the database UseBackendDB provisioned. Calling it first is a
// bug in the test, so it fails the test rather than returning nil.
func (h *Helper) BackendAppDb() *sqlx.DB {
	h.T.Helper()
	if h.db == nil {
		h.T.Fatal("BackendAppDb called before UseBackendDB")
	}
	return h.db
}

// AppConfig is a full app config pointed at this test's database, for tests
// that boot the whole container. The secret is a throwaway.
func (h *Helper) AppConfig() *config.Config {
	h.T.Helper()
	h.BackendAppDb()
	return &config.Config{
		DBHost:    h.dbHost,
		DBPort:    h.dbPort,
		DBUser:    "test",
		DBPass:    "test",
		DBName:    h.dbName,
		JWTSecret: "test-secret-that-is-at-least-32-bytes",
	}
}
