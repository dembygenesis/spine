package org_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dembygenesis/spine/internal/domain/org"
	"github.com/dembygenesis/spine/internal/model"
	"github.com/dembygenesis/spine/internal/testfactory"
	"github.com/dembygenesis/spine/internal/testsuite"
	"github.com/dembygenesis/spine/pkg/apierr"
	"github.com/dembygenesis/spine/pkg/pg"
	"github.com/google/uuid"
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

func doRequest(th *testsuite.Helper, h http.Handler, method, url string, body []byte) (int, []byte) {
	th.T.Helper()
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func orgURL(id string) string {
	return strings.Replace(org.PathOrganization, "{id}", id, 1)
}

func TestAPICreateOrganization(t *testing.T) {
	tests := []struct {
		name       string
		body       func(th *testsuite.Helper) []byte
		wantStatus int
		wantCode   apierr.Code
		wantLogged bool
		assert     func(th *testsuite.Helper, resp []byte)
	}{
		{
			name: "success-creates-the-org",
			body: func(th *testsuite.Helper) []byte {
				return jsonBody(th, model.ReqCreateOrganization{
					Name:    "acme",
					OwnerID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
				})
			},
			wantStatus: http.StatusCreated,
			assert: func(th *testsuite.Helper, resp []byte) {
				var got model.Organization
				require.NoError(th.T, json.Unmarshal(resp, &got))
				var name string
				require.NoError(th.T, th.BackendAppDb().Get(&name, `SELECT name FROM organizations WHERE id = $1`, got.ID))
				require.Equal(th.T, "acme", name)
			},
		},
		{
			name: "error-malformed-json-is-rejected-by-huma",
			body: func(th *testsuite.Helper) []byte {
				return []byte(`{"name":`)
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-name-too-short-is-rejected-by-the-schema",
			body: func(th *testsuite.Helper) []byte {
				return jsonBody(th, model.ReqCreateOrganization{
					Name:    "a",
					OwnerID: uuid.NewString(),
				})
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-name-taken-is-mapped-and-logged",
			body: func(th *testsuite.Helper) []byte {
				testfactory.NewWith(th, testfactory.OrgFactory{
					Subject: &model.Organization{
						Name: "acme",
					},
				})
				return jsonBody(th, model.ReqCreateOrganization{
					Name:    "acme",
					OwnerID: testfactory.New[testfactory.UserFactory](th).Subject.ID,
				})
			},
			wantStatus: http.StatusConflict,
			wantCode:   apierr.CodeConflict,
			wantLogged: true,
		},
		{
			name: "error-unknown-owner-is-mapped-and-logged",
			body: func(th *testsuite.Helper) []byte {
				return jsonBody(th, model.ReqCreateOrganization{
					Name:    "acme",
					OwnerID: uuid.NewString(),
				})
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   apierr.CodeValidation,
			wantLogged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			code, resp := doRequest(th, setupMux(th), http.MethodPost, org.PathOrganizations, tt.body(th))
			require.Equalf(t, tt.wantStatus, code, "resp: %s", resp)
			assertError(th, resp, tt.wantStatus, tt.wantCode, tt.wantLogged)
			if tt.assert != nil {
				tt.assert(th, resp)
			}
		})
	}
}

func TestAPIGetOrganization(t *testing.T) {
	tests := []struct {
		name       string
		id         func(th *testsuite.Helper) string
		wantStatus int
		wantCode   apierr.Code
		wantLogged bool
	}{
		{
			name: "success-found",
			id: func(th *testsuite.Helper) string {
				return testfactory.New[testfactory.OrgFactory](th).Subject.ID
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "error-not-a-uuid-is-rejected-by-the-schema",
			id: func(th *testsuite.Helper) string {
				return "not-a-uuid"
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   apierr.CodeValidation,
		},
		{
			name: "error-not-found-is-mapped-and-logged",
			id: func(th *testsuite.Helper) string {
				return uuid.NewString()
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apierr.CodeNotFound,
			wantLogged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			th := testsuite.New(t)
			t.Cleanup(th.UseBackendDB())

			code, resp := doRequest(th, setupMux(th), http.MethodGet, orgURL(tt.id(th)), nil)
			require.Equalf(t, tt.wantStatus, code, "resp: %s", resp)
			assertError(th, resp, tt.wantStatus, tt.wantCode, tt.wantLogged)
		})
	}
}

// assertError checks the error body and the logging rule: errors the handler
// returns are logged; rejections Huma makes before the handler runs are not.
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
