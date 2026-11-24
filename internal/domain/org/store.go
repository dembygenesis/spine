// Package org is the organizations domain: tenants and the seats users hold in
// them.
package org

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/pkg/pg"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Postgres error codes the store translates into domain errors.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// Store is SQL only. Every method takes the Executor to run on, so the business
// layer decides whether a call is part of a transaction.
type Store struct{}

// NewStore returns a Store. It holds no connection.
func NewStore() *Store { return &Store{} }

// Organizations returns the organizations the filter matches.
func (s *Store) Organizations(ctx context.Context, ex pg.Executor, f model.QueryFilterOrganization) ([]model.Organization, error) {
	q := psql.Select("id", "name", "created_at").From("organizations").OrderBy("created_at")
	if f.ID.Valid {
		q = q.Where(sq.Eq{"id": f.ID.String})
	}
	if f.Name.Valid {
		q = q.Where(sq.Eq{"name": f.Name.String})
	}
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	var out []model.Organization
	if err := sqlx.SelectContext(ctx, ex, &out, query, args...); err != nil {
		return nil, fmt.Errorf("select organizations: %w", err)
	}
	return out, nil
}

// Insert creates an organization.
func (s *Store) Insert(ctx context.Context, ex pg.Executor, in model.InsertOrganization) (*model.Organization, error) {
	query, args, err := psql.Insert("organizations").
		Columns("name").
		Values(in.Name).
		Suffix("RETURNING id, name, created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert: %w", err)
	}

	var out model.Organization
	if err := sqlx.GetContext(ctx, ex, &out, query, args...); err != nil {
		if isPgCode(err, pgUniqueViolation) {
			return nil, model.ErrOrganizationNameTaken
		}
		return nil, fmt.Errorf("insert organization: %w", err)
	}
	return &out, nil
}

// Update changes the fields set in in, and returns the updated row.
func (s *Store) Update(ctx context.Context, ex pg.Executor, in model.UpdateOrganization) (*model.Organization, error) {
	q := psql.Update("organizations").Where(sq.Eq{"id": in.ID}).Suffix("RETURNING id, name, created_at")
	if in.Name.Valid {
		q = q.Set("name", in.Name.String)
	}
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update: %w", err)
	}

	var out model.Organization
	err = sqlx.GetContext(ctx, ex, &out, query, args...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, model.ErrOrganizationNotFound
	case isPgCode(err, pgUniqueViolation):
		return nil, model.ErrOrganizationNameTaken
	case err != nil:
		return nil, fmt.Errorf("update organization: %w", err)
	}
	return &out, nil
}

// InsertMember seats a user in an organization.
func (s *Store) InsertMember(ctx context.Context, ex pg.Executor, in model.InsertOrganizationMember) (*model.OrganizationMember, error) {
	query, args, err := psql.Insert("organization_members").
		Columns("organization_id", "user_id", "role").
		Values(in.OrganizationID, in.UserID, in.Role).
		Suffix("RETURNING id, organization_id, user_id, role, created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert: %w", err)
	}

	var out model.OrganizationMember
	if err := sqlx.GetContext(ctx, ex, &out, query, args...); err != nil {
		if isPgCode(err, pgForeignKeyViolation) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("insert member: %w", err)
	}
	return &out, nil
}

func isPgCode(err error, code string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == code
}
