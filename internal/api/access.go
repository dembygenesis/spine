package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dembygenesis/spine/pkg/pg"

	"github.com/jmoiron/sqlx"
)

// ErrNoSeat is a caller with no membership in the organization they named.
var ErrNoSeat = errors.New("no seat in this organization")

// StorageAccess reads a caller's role in an organization from the database, on
// every request. Nothing about the caller's standing is trusted from the token:
// a removed seat takes effect on the next call, not when the token expires.
//
// imapp2 keeps the same resolver in its platform domain; here it sits at the
// edge so the harness can build it without importing a domain.
type StorageAccess struct {
	db pg.Executor
}

// NewStorageAccess returns a resolver reading from db.
func NewStorageAccess(db pg.Executor) (*StorageAccess, error) {
	if db == nil {
		return nil, errors.New("api: access needs a database")
	}
	return &StorageAccess{db: db}, nil
}

// Role is the role userID holds in orgID, or ErrNoSeat.
func (a *StorageAccess) Role(ctx context.Context, userID, orgID string) (string, error) {
	var role string
	err := sqlx.GetContext(ctx, a.db, &role,
		`SELECT role FROM organization_members WHERE user_id = $1 AND organization_id = $2`, userID, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNoSeat
	}
	if err != nil {
		return "", fmt.Errorf("read seat: %w", err)
	}
	return role, nil
}
