// Package api is the HTTP edge: the mux, the error mapping and the logging
// every domain's routes go through.
package api

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/dembygenesis/spine/pkg/apierr"
	"github.com/dembygenesis/spine/pkg/observe"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// auther turns an Authorization header into a user id.
type auther interface {
	Authenticate(bearer string) (string, error)
}

// accessResolver reads the caller's role in an organization from storage.
type accessResolver interface {
	Role(ctx context.Context, userID, orgID string) (string, error)
}

// permitter answers whether a role may act on a resource.
type permitter interface {
	Can(role, resource, action string) bool
}

// Mux owns the router, the Huma API registered on it, the error mapper every
// domain adds its errors to, and the gates routes are put behind.
type Mux struct {
	router  *http.ServeMux
	huma    huma.API
	mapper  *apierr.Mapper
	logger  observe.Logger
	auther  auther
	access  accessResolver
	catalog permitter
	gated   map[PermissionPair]struct{}
}

// NewParams are the Mux's dependencies, all required.
type NewParams struct {
	Logger  observe.Logger
	Auther  auther
	Access  accessResolver
	Catalog permitter
}

// New builds a Mux with the OpenAPI document named "spine".
func New(p NewParams) (*Mux, error) {
	switch {
	case p.Logger == nil:
		return nil, errors.New("api: logger is required")
	case p.Auther == nil:
		return nil, errors.New("api: auther is required")
	case p.Access == nil:
		return nil, errors.New("api: access resolver is required")
	case p.Catalog == nil:
		return nil, errors.New("api: permission catalog is required")
	}
	router := http.NewServeMux()
	return &Mux{
		router:  router,
		huma:    humago.New(router, huma.DefaultConfig("spine", "0.1.0")),
		mapper:  apierr.NewMapper(),
		logger:  p.Logger,
		auther:  p.Auther,
		access:  p.Access,
		catalog: p.Catalog,
		gated:   map[PermissionPair]struct{}{},
	}, nil
}

// PermissionPair is one (resource, action) a route is gated on.
type PermissionPair struct {
	Resource string
	Action   string
}

// GatedPermissions lists every pair a registered route demands, so a test can
// prove each one exists in the catalog.
func (m *Mux) GatedPermissions() []PermissionPair {
	out := make([]PermissionPair, 0, len(m.gated))
	for p := range m.gated {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Resource+out[i].Action < out[j].Resource+out[j].Action })
	return out
}

// HumaAPI is where domains register their operations.
func (m *Mux) HumaAPI() huma.API { return m.huma }

// RegisterErrors lets a domain add the errors it can return to the mapper.
func (m *Mux) RegisterErrors(register func(*apierr.Mapper)) { register(m.mapper) }

// ServeHTTP makes the Mux an http.Handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) { m.router.ServeHTTP(w, r) }

// SetLogger swaps the logger. Tests only: production wires it in New.
func (m *Mux) SetLogger(l observe.Logger) { m.logger = l }

// Err maps a handler's error to its response and logs it. Every handler returns
// its errors through here, so no error reaches a client unlogged.
func (m *Mux) Err(ctx context.Context, err error) error {
	resp := m.mapper.Map(err)
	m.logger.Error(ctx, resp.Message, err,
		observe.F("status", resp.GetStatus()),
		observe.F("error_code", string(resp.Code)),
	)
	return resp
}
