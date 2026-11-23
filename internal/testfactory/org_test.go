package testfactory_test

import (
	"testing"

	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testfactory"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/stretchr/testify/require"
)

func TestOrganizationMemberFactory(t *testing.T) {
	tests := []struct {
		name   string
		build  func(th *testsuite.Helper) testfactory.OrganizationMemberFactory
		assert func(th *testsuite.Helper, got testfactory.OrganizationMemberFactory)
	}{
		{
			name: "success-builds-its-own-user-and-org",
			build: func(th *testsuite.Helper) testfactory.OrganizationMemberFactory {
				return testfactory.New[testfactory.OrganizationMemberFactory](th)
			},
			assert: func(th *testsuite.Helper, got testfactory.OrganizationMemberFactory) {
				require.True(th.T, got.IsValid())
				require.Equal(th.T, 1, count(th, `SELECT count(*) FROM organizations`))
				require.Equal(th.T, 1, count(th, `SELECT count(*) FROM users`))
			},
		},
		{
			name: "success-defaults-to-the-member-role",
			build: func(th *testsuite.Helper) testfactory.OrganizationMemberFactory {
				return testfactory.New[testfactory.OrganizationMemberFactory](th)
			},
			assert: func(th *testsuite.Helper, got testfactory.OrganizationMemberFactory) {
				var role string
				require.NoError(th.T, th.BackendAppDb().Get(&role,
					`SELECT role FROM organization_members WHERE id = $1`, got.Subject.ID))
				require.Equal(th.T, model.RoleMember, role)
			},
		},
		{
			name: "success-reuses-the-org-it-is-given",
			build: func(th *testsuite.Helper) testfactory.OrganizationMemberFactory {
				org := testfactory.New[testfactory.OrgFactory](th)
				testfactory.NewWith(th, testfactory.OrganizationMemberFactory{
					Subject: &model.OrganizationMember{
						OrganizationID: org.Subject.ID,
					},
				})
				return testfactory.NewWith(th, testfactory.OrganizationMemberFactory{
					Subject: &model.OrganizationMember{
						OrganizationID: org.Subject.ID,
						Role:           model.RoleAdmin,
					},
				})
			},
			assert: func(th *testsuite.Helper, got testfactory.OrganizationMemberFactory) {
				require.Equal(th.T, 1, count(th, `SELECT count(*) FROM organizations`))
				require.Equal(th.T, 2, count(th, `SELECT count(*) FROM organization_members`))
				require.Equal(th.T, model.RoleAdmin, got.Subject.Role)
			},
		},
		{
			name: "success-keeps-the-fields-a-test-sets",
			build: func(th *testsuite.Helper) testfactory.OrganizationMemberFactory {
				org := testfactory.NewWith(th, testfactory.OrgFactory{
					Subject: &model.Organization{
						Name: "acme",
					},
				})
				return testfactory.NewWith(th, testfactory.OrganizationMemberFactory{
					Subject: &model.OrganizationMember{
						OrganizationID: org.Subject.ID,
					},
				})
			},
			assert: func(th *testsuite.Helper, got testfactory.OrganizationMemberFactory) {
				var name string
				require.NoError(th.T, th.BackendAppDb().Get(&name,
					`SELECT o.name FROM organizations o
					 JOIN organization_members m ON m.organization_id = o.id
					 WHERE m.id = $1`, got.Subject.ID))
				require.Equal(th.T, "acme", name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			got := tt.build(th)
			tt.assert(th, got)
		})
	}
}

func count(th *testsuite.Helper, query string) int {
	th.T.Helper()
	var n int
	require.NoError(th.T, th.BackendAppDb().Get(&n, query))
	return n
}
