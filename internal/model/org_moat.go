package model

import "errors"

// ResourceOrganization is the resource the org routes are gated on. The pairs
// it is gated with must have rows in role_permissions.
const ResourceOrganization = "organization"

// Errors the org domain returns. Each one is mapped to an HTTP status where
// the routes are registered; an unmapped error surfaces as a bare 500.
var (
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationNameTaken = errors.New("organization name is taken")
	ErrUserNotFound          = errors.New("user not found")
)
