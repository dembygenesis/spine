package org

import (
	"context"
	"net/http"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/internal/model"

	"github.com/danielgtaylor/huma/v2"
)

// CreateOrganizationInput is the create request.
type CreateOrganizationInput struct {
	Body model.ReqCreateOrganization
}

// GetOrganizationInput names the organization to read.
type GetOrganizationInput struct {
	ID string `path:"id" format:"uuid" doc:"UUID of the organization"`
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
		OwnerID: in.Body.OwnerID,
	})
	if err != nil {
		return nil, hh.mux.Err(ctx, err)
	}
	return &OrganizationOutput{Body: o}, nil
}

func (hh *humaHandlers) organization(ctx context.Context, in *GetOrganizationInput) (*OrganizationOutput, error) {
	o, err := hh.biz.Organization(ctx, in.ID)
	if err != nil {
		return nil, hh.mux.Err(ctx, err)
	}
	return &OrganizationOutput{Body: o}, nil
}

func registerHumaRoutes(humaAPI huma.API, hh *humaHandlers) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-organization",
		Method:        http.MethodPost,
		Path:          PathOrganizations,
		Tags:          []string{"organizations"},
		Summary:       "Create an organization",
		Description:   "Creates an organization and seats owner_id as its first admin. Returns 409 when the name is taken.",
		DefaultStatus: http.StatusCreated,
	}, hh.createOrganization)

	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-organization",
		Method:      http.MethodGet,
		Path:        PathOrganization,
		Tags:        []string{"organizations"},
		Summary:     "Get an organization",
		Description: "Returns one organization by id, or 404.",
	}, hh.organization)
}
