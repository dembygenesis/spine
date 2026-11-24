// Package api is the HTTP edge: the mux, the error mapping and the logging
// every domain's routes go through.
package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/dembygenesis/spine/pkg/apierr"
	"github.com/dembygenesis/spine/pkg/observe"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Mux owns the router, the Huma API registered on it, and the error mapper
// every domain adds its errors to.
type Mux struct {
	router *http.ServeMux
	huma   huma.API
	mapper *apierr.Mapper
	logger observe.Logger
}

// NewParams are the Mux's dependencies.
type NewParams struct {
	Logger observe.Logger
}

// New builds a Mux with the OpenAPI document named "spine".
func New(p NewParams) (*Mux, error) {
	if p.Logger == nil {
		return nil, errors.New("api: logger is required")
	}
	router := http.NewServeMux()
	return &Mux{
		router: router,
		huma:   humago.New(router, huma.DefaultConfig("spine", "0.1.0")),
		mapper: apierr.NewMapper(),
		logger: p.Logger,
	}, nil
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
