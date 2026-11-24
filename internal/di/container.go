// Package di builds the whole app by hand: infrastructure first, then every
// domain's Register. No framework, no reflection, nothing generated: the order
// things are built in is the order you read.
package di

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/internal/config"
	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/migration"
	"github.com/dembygenesis/spine/pkg/auth"
	"github.com/dembygenesis/spine/pkg/observe"
	"github.com/dembygenesis/spine/pkg/pg"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Container is the built app.
type Container struct {
	Handler http.Handler
	JWT     *auth.JWTService
	db      *sqlx.DB
}

// Close releases the database connection.
func (c *Container) Close() error { return c.db.Close() }

// New connects, migrates and wires every domain. Any missing or bad dependency
// is an error here, at startup, never a nil check inside a request.
func New(ctx context.Context, cfg *config.Config, logger observe.Logger) (*Container, error) {
	jwt, err := auth.NewJWTService(cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.ConnectContext(ctx, "postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	c, err := build(ctx, db, cfg, jwt, logger)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func build(ctx context.Context, db *sqlx.DB, cfg *config.Config, jwt *auth.JWTService, logger observe.Logger) (*Container, error) {
	if err := migration.Up(db, cfg.DBName); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	transactor, err := pg.NewTransactor(db)
	if err != nil {
		return nil, err
	}
	access, err := api.NewStorageAccess(db)
	if err != nil {
		return nil, err
	}
	catalog, err := auth.LoadCatalog(ctx, db)
	if err != nil {
		return nil, err
	}
	mux, err := api.New(api.NewParams{
		Logger:  logger,
		Auther:  jwt,
		Access:  access,
		Catalog: catalog,
	})
	if err != nil {
		return nil, err
	}

	if err := org.Register(mux, transactor); err != nil {
		return nil, fmt.Errorf("org: %w", err)
	}
	return &Container{Handler: mux, JWT: jwt, db: db}, nil
}
