package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	usersqlite "github.com/willie68/arcivio/internal/adapter/outbound/identity/sqlite"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/identity"
)

func TestListUsersRequiresAdmin(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := usersqlite.New(st.DB())
	require.NoError(t, err)
	svc := identity.New(repo, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16))

	admin, _, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "admin", Roles: []string{identity.RoleAdmin}})
	require.NoError(t, err)
	clerk, _, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "clerk", FirstName: "Chris", Roles: []string{identity.RoleClerk}})
	require.NoError(t, err)

	h := &usersHandler{ident: svc}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	h.List(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=10", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": clerk.ID, "roles": clerk.Roles},
	}, nil, false))
	h.List(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=1", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": admin.ID, "roles": admin.Roles},
	}, nil, false))
	h.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body UserListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Total)
	assert.Equal(t, 1, body.Page)
	assert.Equal(t, 1, body.PageSize)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "admin", body.Items[0].Username)
	assert.NotContains(t, rec.Body.String(), "password")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=1&sort=username&order=desc", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": admin.ID},
	}, nil, false))
	h.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, "clerk", body.Items[0].Username)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?prefix=chr", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": admin.ID},
	}, nil, false))
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Total)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "clerk", body.Items[0].Username)
}

func TestCreateAndDeleteUser(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := usersqlite.New(st.DB())
	require.NoError(t, err)
	svc := identity.New(repo, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16))
	admin, _, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "admin", Roles: []string{identity.RoleAdmin}})
	require.NoError(t, err)
	h := &usersHandler{ident: svc}

	body, _ := json.Marshal(map[string]any{
		"username":  "neu",
		"firstName": "Neu",
		"lastName":  "User",
		"email":     "neu@example.com",
		"roles":     []string{identity.RoleReader},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": admin.ID},
	}, nil, false))
	h.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created createUserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "neu", created.User.Username)
	assert.NotEmpty(t, created.Password)
	assert.NotContains(t, created.User.ID, created.Password)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+admin.ID, nil)
	req = withUserID(req, admin)
	h.Delete(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+created.User.ID, nil)
	req = withUserID(req, admin)
	h.Delete(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func withUserID(req *http.Request, actor *identity.User) *http.Request {
	ctx := auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": actor.ID},
	}, nil, false)
	route := chi.NewRouteContext()
	route.URLParams.Add("id", req.URL.Path[len("/api/v1/users/"):])
	return req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, route))
}
