// Package model holds the entities every package shares.
package model

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

// Roles a member can hold in an organization. The values are what the
// organization_members.role CHECK constraint accepts.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Organization is a tenant: everything else in the system belongs to one.
type Organization struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// User is a person who can sign in. Which organizations they act in is
// decided by their OrganizationMember rows, not by the user.
type User struct {
	ID        string    `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// OrganizationMember is a user's seat in one organization, with the role
// they hold there.
type OrganizationMember struct {
	ID             string    `db:"id" json:"id"`
	OrganizationID string    `db:"organization_id" json:"organization_id"`
	UserID         string    `db:"user_id" json:"user_id"`
	Role           string    `db:"role" json:"role"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ReqCreateOrganization is the body of a create request. Huma enforces these
// tags before the handler runs and publishes them in the OpenAPI document.
type ReqCreateOrganization struct {
	Name string `json:"name" minLength:"2" maxLength:"64" doc:"Unique name of the organization" example:"acme"`
}

// ReqRenameOrganization is the body of a rename request.
type ReqRenameOrganization struct {
	Name string `json:"name" minLength:"2" maxLength:"64" doc:"The organization's new, unique name" example:"acme-labs"`
}

// InsertOrganization is what a store needs to create an organization.
type InsertOrganization struct {
	Name string
}

// InsertOrganizationMember is what a store needs to seat a user.
type InsertOrganizationMember struct {
	OrganizationID string
	UserID         string
	Role           string
}

// UpdateOrganization changes the organization with ID. Unset fields are left
// as they are.
type UpdateOrganization struct {
	ID   string
	Name null.String
}

// QueryFilterOrganization narrows Organizations. Unset fields do not filter.
type QueryFilterOrganization struct {
	ID   null.String
	Name null.String
}
