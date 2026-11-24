// Package auth issues and checks tokens, and answers whether a role may act on
// a resource.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dembygenesis/spine/pkg/pg"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

// Actions a route can be gated on.
const (
	ActionRead  = "read"
	ActionWrite = "write"
)

// ErrInvalidToken is any token that does not prove who the caller is.
var ErrInvalidToken = errors.New("invalid token")

// ── tokens ────────────────────────────────────────────────────────────────────

// minSecretLen is the shortest HS256 key accepted. A short key is guessable,
// and a service that signs with one should not start.
const minSecretLen = 32

// accessTTL is short on purpose: the token proves identity only. What the caller
// may do is read from storage on every request, so it needs no long life.
const accessTTL = 15 * time.Minute

// JWTService signs and verifies access tokens.
type JWTService struct {
	secret []byte
}

// NewJWTService refuses a missing or short secret.
func NewJWTService(secret string) (*JWTService, error) {
	if len(secret) < minSecretLen {
		return nil, fmt.Errorf("auth: secret must be at least %d bytes", minSecretLen)
	}
	return &JWTService{secret: []byte(secret)}, nil
}

// Issue signs a token naming userID.
func (s *JWTService) Issue(userID string) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
	})
	return t.SignedString(s.secret)
}

// Authenticate turns an Authorization header value into a user id.
func (s *JWTService) Authenticate(bearer string) (string, error) {
	raw, ok := strings.CutPrefix(bearer, "Bearer ")
	if !ok || raw == "" {
		return "", ErrInvalidToken
	}
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

// ── permissions ───────────────────────────────────────────────────────────────

// PermissionRow is one row of role_permissions.
type PermissionRow struct {
	Role     string `db:"role"`
	Resource string `db:"resource"`
	Action   string `db:"action"`
}

// Catalog answers Can(role, resource, action) from the rows it was built with.
type Catalog struct {
	allowed map[PermissionRow]bool
}

// NewCatalog builds a Catalog.
func NewCatalog(rows []PermissionRow) *Catalog {
	c := &Catalog{allowed: make(map[PermissionRow]bool, len(rows))}
	for _, r := range rows {
		c.allowed[r] = true
	}
	return c
}

// LoadCatalog reads role_permissions once, at startup.
func LoadCatalog(ctx context.Context, ex pg.Executor) (*Catalog, error) {
	var rows []PermissionRow
	if err := sqlx.SelectContext(ctx, ex, &rows, `SELECT role, resource, action FROM role_permissions`); err != nil {
		return nil, fmt.Errorf("load permissions: %w", err)
	}
	return NewCatalog(rows), nil
}

// Can reports whether role may perform action on resource. A pair with no row
// is false: the catalog is deny-by-default.
func (c *Catalog) Can(role, resource, action string) bool {
	return c.allowed[PermissionRow{Role: role, Resource: resource, Action: action}]
}

// Has reports whether any role holds the pair at all. A gated route whose pair
// no role holds can never be reached.
func (c *Catalog) Has(resource, action string) bool {
	for r := range c.allowed {
		if r.Resource == resource && r.Action == action {
			return true
		}
	}
	return false
}
