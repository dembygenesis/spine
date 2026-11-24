package org_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testfactory"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/apierr"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/stretchr/testify/require"
)

func setupMux(th *testsuite.Helper) http.Handler {
	th.T.Helper()
	mux := th.SetupAPI()
	tr, err := pg.NewTransactor(th.BackendAppDb())
	require.NoError(th.T, err)
	require.NoError(th.T, org.Register(mux, tr))
	return mux
}

// request is one call. A zero userID sends no Authorization header at all.
type request struct {
	method string
	path   string
	body   []byte
	userID string
	orgID  string
}

func doRequest(th *testsuite.Helper, h http.Handler, r request) (int, []byte) {
	th.T.Helper()
	req := httptest.NewRequest(r.method, r.path, bytes.NewReader(r.body))
	req.Header.Set("Content-Type", "application/json")
	if r.userID != "" {
		th.Fakers.FakeAuther.SetUser(r.userID)
		req.Header.Set("Authorization", "Bearer test")
	}
	if r.orgID != "" {
		req.Header.Set(api.HeaderOrganizationID, r.orgID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	out, err := io.ReadAll(rec.Body)
	require.NoError(th.T, err)
	return rec.Code, out
}

func jsonBody(th *testsuite.Helper, v any) []byte {
	th.T.Helper()
	b, err := json.Marshal(v)
	require.NoError(th.T, err)
	return b
}

// seat creates a user with role in a fresh organization.
func seat(th *testsuite.Helper, role string) model.OrganizationMember {
	th.T.Helper()
	return *testfactory.NewWith(th, testfactory.OrganizationMemberFactory{
		Subject: &model.OrganizationMember{
			Role: role,
		},
	}).Subject
}

func TestAPICreateOrganization(t *testing.T) {
	tests := []struct {
		name       string
		request    func(th *testsuite.Helper) request
		wantStatus int
		wantCode   apierr.Code
		wantLogged bool
		assert     func(th *testsuite.Helper, resp []byte)
	}{
		{
			name: "success-the-caller-becomes-its-admin",
			request: func(th *testsuite.Helper) request {
				return request{
					userID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
					body:   jsonBody(th, model.ReqCreateOrganization{Name: "acme"}),
				}
			},
			wantStatus: http.StatusCreated,
			assert: func(th *testsuite.Helper, resp []byte) {
				var got model.Organization
				require.NoError(th.T, json.Unmarshal(resp, &got))
				var role string
				require.NoError(th.T, th.BackendAppDb().Get(&role,
					`SELECT role FROM organization_members WHERE organization_id = $1`, got.ID))
				require.Equal(th.T, model.RoleAdmin, role)
			},
		},
		{
			name: "error-no-token",
			request: func(th *testsuite.Helper) request {
				return request{
					body: jsonBody(th, model.ReqCreateOrganization{Name: "acme"}),
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   apierr.CodeUnauthorized,
			wantLogged: true,
		},
		{
			name: "error-malformed-json-is-rejected-by-huma",
			request: func(th *testsuite.Helper) request {
				return request{
					userID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
					body:   []byte(`{"name":`),
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-name-too-short-is-rejected-by-the-schema",
			request: func(th *testsuite.Helper) request {
				return request{
					userID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
					body:   jsonBody(th, model.ReqCreateOrganization{Name: "a"}),
				}
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-name-taken-is-mapped-and-logged",
			request: func(th *testsuite.Helper) request {
				testfactory.NewWith(th, testfactory.OrgFactory{
					Subject: &model.Organization{
						Name: "acme",
					},
				})
				return request{
					userID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
					body:   jsonBody(th, model.ReqCreateOrganization{Name: "acme"}),
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   apierr.CodeConflict,
			wantLogged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			r := tt.request(th)
			r.method, r.path = http.MethodPost, org.PathOrganizations
			code, resp := doRequest(th, setupMux(th), r)
			require.Equalf(t, tt.wantStatus, code, "resp: %s", resp)
			assertError(th, resp, tt.wantStatus, tt.wantCode, tt.wantLogged)
			if tt.assert != nil {
				tt.assert(th, resp)
			}
		})
	}
}

func TestAPIOrganization(t *testing.T) {
	tests := []struct {
		name       string
		request    func(th *testsuite.Helper) request
		wantStatus int
		wantCode   apierr.Code
	}{
		{
			name: "success-a-member-reads-their-org",
			request: func(th *testsuite.Helper) request {
				m := seat(th, model.RoleMember)
				return request{
					userID: m.UserID,
					orgID:  m.OrganizationID,
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "error-no-token",
			request: func(th *testsuite.Helper) request {
				return request{
					orgID: seat(th, model.RoleMember).OrganizationID,
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   apierr.CodeUnauthorized,
		},
		{
			name: "error-no-organization-header",
			request: func(th *testsuite.Helper) request {
				return request{
					userID: seat(th, model.RoleMember).UserID,
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-another-tenants-org-is-forbidden",
			request: func(th *testsuite.Helper) request {
				mine := seat(th, model.RoleAdmin)
				theirs := seat(th, model.RoleAdmin)
				return request{
					userID: mine.UserID,
					orgID:  theirs.OrganizationID,
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   apierr.CodeForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			r := tt.request(th)
			r.method, r.path = http.MethodGet, org.PathOrganization
			code, resp := doRequest(th, setupMux(th), r)
			require.Equalf(t, tt.wantStatus, code, "resp: %s", resp)
			assertError(th, resp, tt.wantStatus, tt.wantCode, true)
		})
	}
}

func TestAPIRenameOrganization(t *testing.T) {
	tests := []struct {
		name       string
		request    func(th *testsuite.Helper) request
		wantStatus int
		wantCode   apierr.Code
		assert     func(th *testsuite.Helper, r request)
	}{
		{
			name: "success-an-admin-renames",
			request: func(th *testsuite.Helper) request {
				m := seat(th, model.RoleAdmin)
				return request{
					userID: m.UserID,
					orgID:  m.OrganizationID,
					body:   jsonBody(th, model.ReqRenameOrganization{Name: "acme-labs"}),
				}
			},
			wantStatus: http.StatusOK,
			assert: func(th *testsuite.Helper, r request) {
				var name string
				require.NoError(th.T, th.BackendAppDb().Get(&name, `SELECT name FROM organizations WHERE id = $1`, r.orgID))
				require.Equal(th.T, "acme-labs", name)
			},
		},
		{
			name: "error-a-member-may-not-rename",
			request: func(th *testsuite.Helper) request {
				m := seat(th, model.RoleMember)
				return request{
					userID: m.UserID,
					orgID:  m.OrganizationID,
					body:   jsonBody(th, model.ReqRenameOrganization{Name: "acme-labs"}),
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   apierr.CodeForbidden,
			assert: func(th *testsuite.Helper, r request) {
				var n int
				require.NoError(th.T, th.BackendAppDb().Get(&n, `SELECT count(*) FROM organizations WHERE name = 'acme-labs'`))
				require.Equal(th.T, 0, n, "a refused rename must not change the row")
			},
		},
		{
			name: "error-name-taken",
			request: func(th *testsuite.Helper) request {
				testfactory.NewWith(th, testfactory.OrgFactory{
					Subject: &model.Organization{
						Name: "acme-labs",
					},
				})
				m := seat(th, model.RoleAdmin)
				return request{
					userID: m.UserID,
					orgID:  m.OrganizationID,
					body:   jsonBody(th, model.ReqRenameOrganization{Name: "acme-labs"}),
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   apierr.CodeConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			r := tt.request(th)
			r.method, r.path = http.MethodPatch, org.PathOrganization
			code, resp := doRequest(th, setupMux(th), r)
			require.Equalf(t, tt.wantStatus, code, "resp: %s", resp)
			assertError(th, resp, tt.wantStatus, tt.wantCode, true)
			if tt.assert != nil {
				tt.assert(th, r)
			}
		})
	}
}

// assertError checks the error body and the logging rule: errors from the
// handler and from the auth middleware are logged; rejections Huma makes on its
// own, bad JSON and failed schema tags, are not.
func assertError(th *testsuite.Helper, resp []byte, status int, code apierr.Code, logged bool) {
	th.T.Helper()
	if code == "" {
		require.Nil(th.T, th.Fakers.TestLogger.LastError(), "a success must not log an error")
		return
	}
	var errResp apierr.HumaError
	require.NoError(th.T, json.Unmarshal(resp, &errResp))
	require.Equal(th.T, code, errResp.Code)

	last := th.Fakers.TestLogger.LastError()
	if !logged {
		require.Nil(th.T, last, "Huma rejected this before the handler; nothing should be logged")
		return
	}
	require.NotNil(th.T, last, "expected the error to be logged")
	require.Equal(th.T, status, last.Fields["status"])
}
