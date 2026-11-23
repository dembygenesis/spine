package testfactory

import (
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ── OrgFactory ────────────────────────────────────────────────────────────────

// OrgFactory builds an organization. It has no foreign keys.
type OrgFactory struct {
	th      *testsuite.Helper
	Subject *model.Organization
}

func (f OrgFactory) New(th *testsuite.Helper) OrgFactory {
	f.th = th
	if f.Subject == nil {
		f.Subject = &model.Organization{}
	}
	return f
}

func (f OrgFactory) EnsureFKDeps() OrgFactory { return f }

// SetRequiredFields gives the name a random suffix: names are UNIQUE, and two
// orgs in one test must not collide.
func (f OrgFactory) SetRequiredFields() OrgFactory {
	if f.Subject.Name == "" {
		f.Subject.Name = "org-" + uuid.NewString()[:8]
	}
	return f
}

func (f OrgFactory) Save() OrgFactory {
	q, args, err := psql.Insert("organizations").
		Columns("name").
		Values(f.Subject.Name).
		Suffix("RETURNING id, name, created_at").
		ToSql()
	require.NoError(f.th.T, err)
	require.NoError(f.th.T, f.th.BackendAppDb().Get(f.Subject, q, args...))
	return f
}

func (f OrgFactory) IsValid() bool {
	return f.Subject != nil && f.Subject.ID != ""
}

// ── UserFactory ───────────────────────────────────────────────────────────────

// UserFactory builds a user. It has no foreign keys.
type UserFactory struct {
	th      *testsuite.Helper
	Subject *model.User
}

func (f UserFactory) New(th *testsuite.Helper) UserFactory {
	f.th = th
	if f.Subject == nil {
		f.Subject = &model.User{}
	}
	return f
}

func (f UserFactory) EnsureFKDeps() UserFactory { return f }

func (f UserFactory) SetRequiredFields() UserFactory {
	if f.Subject.Email == "" {
		f.Subject.Email = "user-" + uuid.NewString()[:8] + "@example.test"
	}
	return f
}

func (f UserFactory) Save() UserFactory {
	q, args, err := psql.Insert("users").
		Columns("email").
		Values(f.Subject.Email).
		Suffix("RETURNING id, email, created_at").
		ToSql()
	require.NoError(f.th.T, err)
	require.NoError(f.th.T, f.th.BackendAppDb().Get(f.Subject, q, args...))
	return f
}

func (f UserFactory) IsValid() bool {
	return f.Subject != nil && f.Subject.ID != ""
}

// ── OrganizationMemberFactory ─────────────────────────────────────────────────

// OrganizationMemberFactory builds a user's seat in an organization, creating
// the user and the organization when the test did not supply them.
type OrganizationMemberFactory struct {
	th      *testsuite.Helper
	Subject *model.OrganizationMember
}

func (f OrganizationMemberFactory) New(th *testsuite.Helper) OrganizationMemberFactory {
	f.th = th
	if f.Subject == nil {
		f.Subject = &model.OrganizationMember{}
	}
	return f
}

// EnsureFKDeps creates the parents a membership cannot exist without. This is
// what lets a test that only cares about roles stay one line long.
func (f OrganizationMemberFactory) EnsureFKDeps() OrganizationMemberFactory {
	if f.Subject.OrganizationID == "" {
		f.Subject.OrganizationID = New[OrgFactory](f.th).Subject.ID
	}
	if f.Subject.UserID == "" {
		f.Subject.UserID = New[UserFactory](f.th).Subject.ID
	}
	return f
}

// SetRequiredFields defaults to member: a test that forgets to pick a role
// should not quietly get admin.
func (f OrganizationMemberFactory) SetRequiredFields() OrganizationMemberFactory {
	if f.Subject.Role == "" {
		f.Subject.Role = model.RoleMember
	}
	return f
}

func (f OrganizationMemberFactory) Save() OrganizationMemberFactory {
	q, args, err := psql.Insert("organization_members").
		Columns("organization_id", "user_id", "role").
		Values(f.Subject.OrganizationID, f.Subject.UserID, f.Subject.Role).
		Suffix("RETURNING id, organization_id, user_id, role, created_at").
		ToSql()
	require.NoError(f.th.T, err)
	require.NoError(f.th.T, f.th.BackendAppDb().Get(f.Subject, q, args...))
	return f
}

func (f OrganizationMemberFactory) IsValid() bool {
	return f.Subject != nil && f.Subject.ID != ""
}
