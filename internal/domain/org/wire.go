package org

import (
	"fmt"
	"net/http"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/pkg/apierr"
	"github.com/dembygenesis/spine/pkg/pg"
)

// Register builds the org domain and puts its errors and routes on the mux.
func Register(mux *api.Mux, transactor *pg.Transactor) error {
	biz, err := NewBusiness(NewBusinessParams{
		Transactor: transactor,
		Store:      NewStore(),
	})
	if err != nil {
		return fmt.Errorf("org business: %w", err)
	}

	mux.RegisterErrors(func(m *apierr.Mapper) {
		m.Register(model.ErrOrganizationNotFound, http.StatusNotFound, apierr.CodeNotFound)
		m.Register(model.ErrOrganizationNameTaken, http.StatusConflict, apierr.CodeConflict)
		m.Register(model.ErrUserNotFound, http.StatusUnprocessableEntity, apierr.CodeValidation)
	})
	registerHumaRoutes(mux.HumaAPI(), &humaHandlers{biz: biz, mux: mux})
	return nil
}
