package org

import (
	"context"
	"errors"
	"fmt"

	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/pkg/pg"

	"gopkg.in/guregu/null.v4"
)

// orgPersister is the slice of the store the business layer uses.
type orgPersister interface {
	Organizations(ctx context.Context, ex pg.Executor, f model.QueryFilterOrganization) ([]model.Organization, error)
	Insert(ctx context.Context, ex pg.Executor, in model.InsertOrganization) (*model.Organization, error)
	InsertMember(ctx context.Context, ex pg.Executor, in model.InsertOrganizationMember) (*model.OrganizationMember, error)
}

// Business owns the rules and the transaction boundaries.
type Business struct {
	transactor *pg.Transactor
	store      orgPersister
}

// NewBusinessParams are Business's dependencies, all required.
type NewBusinessParams struct {
	Transactor *pg.Transactor
	Store      orgPersister
}

// NewBusiness fails at construction when a dependency is missing, so no method
// ever has to check.
func NewBusiness(p NewBusinessParams) (*Business, error) {
	if p.Transactor == nil {
		return nil, errors.New("org: transactor is required")
	}
	if p.Store == nil {
		return nil, errors.New("org: store is required")
	}
	return &Business{transactor: p.Transactor, store: p.Store}, nil
}

// CreateOrganizationParams is what creating an organization needs.
type CreateOrganizationParams struct {
	Name    string
	OwnerID string
}

// CreateOrganization creates an organization and seats its owner as admin. Both
// rows land or neither does: an organization nobody can administer is worse than
// no organization.
func (b *Business) CreateOrganization(ctx context.Context, p CreateOrganizationParams) (*model.Organization, error) {
	tx, err := b.transactor.Begin()
	if err != nil {
		return nil, err
	}
	defer b.transactor.Rollback(tx)

	o, err := b.store.Insert(ctx, tx, model.InsertOrganization{Name: p.Name})
	if err != nil {
		return nil, err
	}
	if _, err := b.store.InsertMember(ctx, tx, model.InsertOrganizationMember{
		OrganizationID: o.ID,
		UserID:         p.OwnerID,
		Role:           model.RoleAdmin,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return o, nil
}

// Organization returns the one organization with id.
func (b *Business) Organization(ctx context.Context, id string) (*model.Organization, error) {
	orgs, err := b.store.Organizations(ctx, b.transactor.DB(), model.QueryFilterOrganization{
		ID: null.StringFrom(id),
	})
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, model.ErrOrganizationNotFound
	}
	return &orgs[0], nil
}
