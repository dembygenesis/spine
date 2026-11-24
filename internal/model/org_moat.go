package model

import "errors"

// Errors the org domain returns. Each one is mapped to an HTTP status where
// the routes are registered; an unmapped error surfaces as a bare 500.
var (
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationNameTaken = errors.New("organization name is taken")
	ErrUserNotFound          = errors.New("user not found")
)
