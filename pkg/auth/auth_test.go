package auth_test

import (
	"strings"
	"testing"

	"github.com/dembygenesis/spine/pkg/auth"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret-that-is-at-least-32-bytes"

func TestJWTServiceAuthenticate(t *testing.T) {
	signer, err := auth.NewJWTService(secret)
	require.NoError(t, err)
	other, err := auth.NewJWTService(strings.Repeat("x", 32))
	require.NoError(t, err)

	tests := []struct {
		name    string
		bearer  func(t *testing.T) string
		want    string
		wantErr error
	}{
		{
			name: "success-returns-the-subject",
			bearer: func(t *testing.T) string {
				tok, err := signer.Issue("user-1")
				require.NoError(t, err)
				return "Bearer " + tok
			},
			want: "user-1",
		},
		{
			name: "error-missing-bearer-prefix",
			bearer: func(t *testing.T) string {
				tok, err := signer.Issue("user-1")
				require.NoError(t, err)
				return tok
			},
			wantErr: auth.ErrInvalidToken,
		},
		{
			name: "error-signed-with-another-secret",
			bearer: func(t *testing.T) string {
				tok, err := other.Issue("user-1")
				require.NoError(t, err)
				return "Bearer " + tok
			},
			wantErr: auth.ErrInvalidToken,
		},
		{
			name: "error-tampered-payload",
			bearer: func(t *testing.T) string {
				tok, err := signer.Issue("user-1")
				require.NoError(t, err)
				parts := strings.Split(tok, ".")
				parts[1] = parts[1][:len(parts[1])-2] + "AA"
				return "Bearer " + strings.Join(parts, ".")
			},
			wantErr: auth.ErrInvalidToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := signer.Authenticate(tt.bearer(t))
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewJWTService(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{
			name:   "success-32-bytes",
			secret: strings.Repeat("x", 32),
		},
		{
			name:    "error-short-secret-refuses-to-start",
			secret:  "short",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := auth.NewJWTService(tt.secret)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestCatalogCan(t *testing.T) {
	c := auth.NewCatalog([]auth.PermissionRow{
		{Role: "admin", Resource: "organization", Action: "write"},
	})
	tests := []struct {
		name     string
		role     string
		resource string
		action   string
		want     bool
	}{
		{
			name:     "success-a-listed-pair-is-allowed",
			role:     "admin",
			resource: "organization",
			action:   "write",
			want:     true,
		},
		{
			name:     "success-another-role-is-denied",
			role:     "member",
			resource: "organization",
			action:   "write",
		},
		{
			name:     "success-an-unlisted-pair-is-denied-by-default",
			role:     "admin",
			resource: "organization",
			action:   "delete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, c.Can(tt.role, tt.resource, tt.action))
		})
	}
}
