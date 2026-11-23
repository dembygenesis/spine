package testsuite

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

// Helper is the one value a test carries around: its *testing.T and, once
// provisioned, its database. Anything that needs both (fixtures, factories,
// assertions) takes a *Helper instead of two arguments that must stay paired.
type Helper struct {
	T *testing.T

	db *sqlx.DB
}

// New starts a Helper for one test or subtest.
func New(t *testing.T) *Helper {
	t.Helper()
	return &Helper{T: t}
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
