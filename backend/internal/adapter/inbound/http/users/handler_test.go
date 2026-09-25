package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/identity"
)

func TestListUsersRequiresAdmin(t *testing.T) {
	ident := newMockIdentityService(t)
	admin := identity.User{ID: "admin-id", Username: "admin", Roles: []string{identity.RoleAdmin}}
	clerk := identity.User{ID: "clerk-id", Username: "clerk", FirstName: "Chris", Roles: []string{identity.RoleClerk}}
	ident.EXPECT().GetByID(mock.Anything, clerk.ID).Return(&clerk, nil).Once()
	ident.EXPECT().GetByID(mock.Anything, admin.ID).Return(&admin, nil).Times(3)
	ident.EXPECT().ListUsers(mock.Anything, 0, 1, "", false, "").
		Return([]identity.User{admin}, 2, nil).Once()
	ident.EXPECT().ListUsers(mock.Anything, 0, 1, "username", true, "").
		Return([]identity.User{clerk}, 2, nil).Once()
	ident.EXPECT().ListUsers(mock.Anything, 0, 10, "", false, "chr").
		Return([]identity.User{clerk}, 1, nil).Once()

	h := &Handler{ident: ident}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	h.List(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=10", nil)
	req = asUser(req, clerk.ID)
	h.List(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=1", nil)
	req = asUser(req, admin.ID)
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
	req = asUser(req, admin.ID)
	h.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, "clerk", body.Items[0].Username)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?prefix=chr", nil)
	req = asUser(req, admin.ID)
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Total)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "clerk", body.Items[0].Username)
}

func TestCreateAndDeleteUser(t *testing.T) {
	ident := newMockIdentityService(t)
	admin := identity.User{ID: "admin-id", Username: "admin", Roles: []string{identity.RoleAdmin}}
	createdUser := identity.User{ID: "neu-id", Username: "neu", Roles: []string{identity.RoleReader}}
	ident.EXPECT().GetByID(mock.Anything, admin.ID).Return(&admin, nil).Times(3)
	ident.EXPECT().CreateUser(mock.Anything, identity.NewUser{
		Username:  "neu",
		FirstName: "Neu",
		LastName:  "User",
		Email:     "neu@example.com",
		Roles:     []string{identity.RoleReader},
	}).Return(&createdUser, "temp-secret", nil).Once()
	ident.EXPECT().DeleteUser(mock.Anything, admin.ID, admin.ID).Return(identity.ErrDeleteSelf).Once()
	ident.EXPECT().DeleteUser(mock.Anything, admin.ID, createdUser.ID).Return(nil).Once()

	h := &Handler{ident: ident}

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
	req = asUser(req, admin.ID)
	h.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created createUserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "neu", created.User.Username)
	assert.Equal(t, "temp-secret", created.Password)
	assert.NotContains(t, created.User.ID, created.Password)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+admin.ID, nil)
	req = withUserID(req, admin.ID)
	h.Delete(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+createdUser.ID, nil)
	req = withUserID(req, admin.ID)
	h.Delete(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUpdateAndResetUser(t *testing.T) {
	ident := newMockIdentityService(t)
	admin := identity.User{ID: "admin-id", Username: "admin", Roles: []string{identity.RoleAdmin}}
	updatedUser := identity.User{
		ID:        "clerk-id",
		Username:  "klara",
		FirstName: "Klara",
		Roles:     []string{identity.RoleReader},
	}
	resetUser := updatedUser
	resetUser.MustChangePassword = true
	ident.EXPECT().GetByID(mock.Anything, admin.ID).Return(&admin, nil).Times(3)
	ident.EXPECT().UpdateUser(mock.Anything, "clerk-id", identity.UserPatch{
		Username:  "klara",
		FirstName: "Klara",
		LastName:  "Berg",
		Email:     "klara@example.com",
		Roles:     []string{identity.RoleReader},
	}).Return(&updatedUser, nil).Once()
	ident.EXPECT().UpdateUser(mock.Anything, admin.ID, identity.UserPatch{
		Username: "admin",
		Roles:    []string{identity.RoleReader},
	}).Return(nil, identity.ErrLastAdmin).Once()
	ident.EXPECT().ResetPassword(mock.Anything, "clerk-id").
		Return(&resetUser, "reset-secret", nil).Once()

	h := &Handler{ident: ident}

	body, _ := json.Marshal(map[string]any{
		"username":  "klara",
		"firstName": "Klara",
		"lastName":  "Berg",
		"email":     "klara@example.com",
		"roles":     []string{identity.RoleReader},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/clerk-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, admin.ID)
	h.Update(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var updated UserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, "klara", updated.Username)
	assert.Equal(t, "Klara", updated.FirstName)
	assert.Equal(t, []string{identity.RoleReader}, updated.Roles)
	assert.NotContains(t, rec.Body.String(), "password")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+admin.ID, bytes.NewReader([]byte(`{"username":"admin","roles":["reader"]}`)))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, admin.ID)
	h.Update(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/clerk-id/password-reset", nil)
	req = withUserID(req, admin.ID)
	h.ResetPassword(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var reset createUserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &reset))
	assert.Equal(t, "reset-secret", reset.Password)
	assert.True(t, reset.User.MustChangePassword)
}

func asUser(req *http.Request, userID string) *http.Request {
	return req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": userID},
	}, nil, false))
}

func withUserID(req *http.Request, actorID string) *http.Request {
	req = asUser(req, actorID)
	id := strings.TrimPrefix(req.URL.Path, "/api/v1/users/")
	if i := strings.IndexByte(id, '/'); i >= 0 {
		id = id[:i]
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
}
