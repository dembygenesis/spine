package org_test

import (
	"context"
	"testing"

	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testfactory"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newBusiness(th *testsuite.Helper) *org.Business {
	th.T.Helper()
	tr, err := pg.NewTransactor(th.BackendAppDb())
	require.NoError(th.T, err)
	biz, err := org.NewBusiness(org.NewBusinessParams{
		Transactor: tr,
		Store:      org.NewStore(),
	})
	require.NoError(th.T, err)
	return biz
}

func TestBusinessCreateOrganization(t *testing.T) {
	tests := []struct {
		name    string
		params  func(th *testsuite.Helper) org.CreateOrganizationParams
		wantErr error
		assert  func(th *testsuite.Helper, got *model.Organization)
	}{
		{
			name: "success-creates-the-org-and-seats-the-owner-as-admin",
			params: func(th *testsuite.Helper) org.CreateOrganizationParams {
				return org.CreateOrganizationParams{
					Name:    "acme",
					OwnerID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
				}
			},
			assert: func(th *testsuite.Helper, got *model.Organization) {
				var role string
				require.NoError(th.T, th.BackendAppDb().Get(&role,
					`SELECT role FROM organization_members WHERE organization_id = $1`, got.ID))
				require.Equal(th.T, model.RoleAdmin, role)
			},
		},
		{
			name: "error-name-taken",
			params: func(th *testsuite.Helper) org.CreateOrganizationParams {
				testfactory.NewWith(th, testfactory.OrgFactory{
					Subject: &model.Organization{
						Name: "acme",
					},
				})
				return org.CreateOrganizationParams{
					Name:    "acme",
					OwnerID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
				}
			},
			wantErr: model.ErrOrganizationNameTaken,
		},
		{
			name: "error-unknown-owner-leaves-no-org-behind",
			params: func(th *testsuite.Helper) org.CreateOrganizationParams {
				return org.CreateOrganizationParams{
					Name:    "acme",
					OwnerID: uuid.NewString(),
				}
			},
			wantErr: model.ErrUserNotFound,
			assert: func(th *testsuite.Helper, _ *model.Organization) {
				var n int
				require.NoError(th.T, th.BackendAppDb().Get(&n, `SELECT count(*) FROM organizations`))
				require.Equal(th.T, 0, n, "the org insert must roll back with the failed seat")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())
			biz := newBusiness(th)

			got, err := biz.CreateOrganization(context.Background(), tt.params(th))
			require.ErrorIs(t, err, tt.wantErr)
			if tt.assert != nil {
				tt.assert(th, got)
			}
		})
	}
}

func TestBusinessOrganization(t *testing.T) {
	tests := []struct {
		name    string
		id      func(th *testsuite.Helper) string
		wantErr error
	}{
		{
			name: "success-found",
			id: func(th *testsuite.Helper) string {
				return testfactory.New[testfactory.OrgFactory](th).Subject.ID
			},
		},
		{
			name: "error-not-found",
			id: func(th *testsuite.Helper) string {
				return uuid.NewString()
			},
			wantErr: model.ErrOrganizationNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			id := tt.id(th)
			got, err := newBusiness(th).Organization(context.Background(), id)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				require.Equal(t, id, got.ID)
			}
		})
	}
}
