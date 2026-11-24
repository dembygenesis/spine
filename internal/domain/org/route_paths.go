package org

// Route paths for the org domain, in net/http's {param} syntax. Tests build
// URLs from these, never from inline strings.
const (
	PathOrganizations = "/organizations"
	PathOrganization  = "/organizations/{id}"
)
