package org

import (
	"context"
	"net/http"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/internal/model"

	"github.com/danielgtaylor/huma/v2"
)

// CreateOrganizationInput is the create request. The owner is the caller, so
// it is not in the body: a body field is one a client can set to anyone.
type CreateOrganizationInput struct {
	Body model.ReqCreateOrganization
}

// CurrentOrganizationInput names the organization through the header.
type CurrentOrganizationInput struct {
	OrganizationID string `header:"X-Organization-Id" required:"true" format:"uuid" doc:"The organization to act in"`
}

// RenameOrganizationInput renames the current organization.
type RenameOrganizationInput struct {
	OrganizationID string `header:"X-Organization-Id" required:"true" format:"uuid" doc:"The organization to act in"`
	Body           model.ReqRenameOrganization
}

// OrganizationOutput is an organization in a response body.
type OrganizationOutput struct {
	Body *model.Organization
}

// humaHandlers translate a request into one business call and its error
// through mux.Err. No logic lives here.
type humaHandlers struct {
	biz *Business
	mux *api.Mux
}

func (hh *humaHandlers) createOrganization(ctx context.Context, in *CreateOrganizationInput) (*OrganizationOutput, error) {
	o, err := hh.biz.CreateOrganization(ctx, CreateOrganizationParams{
		Name:    in.Body.Name,
		OwnerID: api.CallerFromContext(ctx).UserID,
	})
	if err != nil {
		return nil, hh.mux.Err(ctx, err)
	}
	return &OrganizationOutput{Body: o}, nil
}

// The tenant comes from the caller MwCan resolved, never from the input: the
// header was already checked against the caller's seat.
func (hh *humaHandlers) organization(ctx context.Context, _ *CurrentOrganizationInput) (*OrganizationOutput, error) {
	o, err := hh.biz.Organization(ctx, api.CallerFromContext(ctx).OrganizationID)
	if err != nil {
		return nil, hh.mux.Err(ctx, err)
	}
	return &OrganizationOutput{Body: o}, nil
}

func (hh *humaHandlers) renameOrganization(ctx context.Context, in *RenameOrganizationInput) (*OrganizationOutput, error) {
	o, err := hh.biz.RenameOrganization(ctx, RenameOrganizationParams{
		ID:   api.CallerFromContext(ctx).OrganizationID,
		Name: in.Body.Name,
	})
	if err != nil {
		return nil, hh.mux.Err(ctx, err)
	}
	return &OrganizationOutput{Body: o}, nil
}

// routeGates are the middlewares wire.go builds and the routes are put behind.
type routeGates struct {
	auth  api.Middleware
	read  api.Middleware
	write api.Middleware
}

func registerHumaRoutes(humaAPI huma.API, hh *humaHandlers, g routeGates) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-organization",
		Method:        http.MethodPost,
		Path:          PathOrganizations,
		Tags:          []string{"organizations"},
		Summary:       "Create an organization",
		Description:   "Creates an organization and seats the caller as its first admin. Returns 409 when the name is taken.",
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{g.auth},
	}, hh.createOrganization)

	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-organization",
		Method:      http.MethodGet,
		Path:        PathOrganization,
		Tags:        []string{"organizations"},
		Summary:     "Get the current organization",
		Description: "Returns the organization named by X-Organization-Id. 403 when the caller holds no seat in it.",
		Middlewares: huma.Middlewares{g.auth, g.read},
	}, hh.organization)

	huma.Register(humaAPI, huma.Operation{
		OperationID: "rename-organization",
		Method:      http.MethodPatch,
		Path:        PathOrganization,
		Tags:        []string{"organizations"},
		Summary:     "Rename the current organization",
		Description: "Renames the organization named by X-Organization-Id. Admins only; 409 when the name is taken.",
		Middlewares: huma.Middlewares{g.auth, g.write},
	}, hh.renameOrganization)
}
