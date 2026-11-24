package org

// Route paths for the org domain. Tests build URLs from these, never from
// inline strings.
const (
	PathOrganizations = "/organizations"
	// PathOrganization is the caller's current organization, named by the
	// X-Organization-Id header rather than the URL.
	PathOrganization = "/organization"
)
