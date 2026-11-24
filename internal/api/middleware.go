package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/dembygenesis/spine/pkg/observe"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// HeaderOrganizationID names the organization a request acts in. The tenant
// comes from here, never from a request body: a body field is one the client
// can set to anything.
const HeaderOrganizationID = "X-Organization-Id"

// Middleware is a Huma operation middleware.
type Middleware = func(ctx huma.Context, next func(huma.Context))

// Caller is who is calling, and once MwCan has run, where and as what.
type Caller struct {
	UserID         string
	OrganizationID string
	Role           string
}

type callerKey struct{}

// CallerFromContext is the authenticated caller, or nil on an ungated route.
func CallerFromContext(ctx context.Context) *Caller {
	c, _ := ctx.Value(callerKey{}).(*Caller)
	return c
}

// MwAuth requires a valid token and puts the caller on the context. It proves
// identity only; what the caller may do is MwCan's job.
func (m *Mux) MwAuth() Middleware {
	return func(ctx huma.Context, next func(huma.Context)) {
		userID, err := m.auther.Authenticate(ctx.Header("Authorization"))
		if err != nil {
			m.reject(ctx, http.StatusUnauthorized, "authentication required", err)
			return
		}
		next(huma.WithValue(ctx, callerKey{}, &Caller{UserID: userID}))
	}
}

// MwCan admits a caller whose role in the X-Organization-Id organization may
// perform action on resource. The role is read from storage on every request.
//
// The pair is recorded as it is gated, so the route that enforces a permission
// is also the list a test checks against the catalog.
func (m *Mux) MwCan(resource, action string) Middleware {
	m.gated[PermissionPair{Resource: resource, Action: action}] = struct{}{}
	return func(ctx huma.Context, next func(huma.Context)) {
		caller := CallerFromContext(ctx.Context())
		if caller == nil {
			m.reject(ctx, http.StatusUnauthorized, "authentication required", errors.New("MwCan without MwAuth"))
			return
		}
		orgID := ctx.Header(HeaderOrganizationID)
		if _, err := uuid.Parse(orgID); err != nil {
			m.reject(ctx, http.StatusBadRequest, HeaderOrganizationID+" must name an organization", err)
			return
		}

		role, err := m.access.Role(ctx.Context(), caller.UserID, orgID)
		if errors.Is(err, ErrNoSeat) {
			m.reject(ctx, http.StatusForbidden, "no access to this organization", err)
			return
		}
		if err != nil {
			m.reject(ctx, http.StatusInternalServerError, "internal error", err)
			return
		}
		if !m.catalog.Can(role, resource, action) {
			m.reject(ctx, http.StatusForbidden, "not allowed", errors.New(role+" may not "+action+" "+resource))
			return
		}

		caller.OrganizationID = orgID
		caller.Role = role
		next(ctx)
	}
}

// reject writes an error response from middleware and logs it, the same as a
// handler's error through Err.
func (m *Mux) reject(ctx huma.Context, status int, msg string, err error) {
	m.logger.Error(ctx.Context(), msg, err, observe.F("status", status))
	_ = huma.WriteErr(m.huma, ctx, status, msg) // the response is already committed; nothing to do on failure
}
