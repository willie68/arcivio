package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	usersqlite "github.com/willie68/arcivio/internal/adapter/outbound/identity/sqlite"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/domain/roles"
)

func TestUpdateOwnProfile(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := usersqlite.New(st.DB())
	require.NoError(t, err)
	svc := identity.New(repo, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16), roles.NewService())
	user, _, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "reader", Roles: []string{roles.RoleReader}})
	require.NoError(t, err)
	h := &meHandler{ident: svc}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader([]byte(`{"firstName":"Ida","lastName":"Horn","email":"ida@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": user.ID},
	}, nil, false))
	h.UpdateProfile(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var me MeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.Equal(t, "reader", me.Username)
	assert.Equal(t, "Ida", me.FirstName)
	assert.Equal(t, "Horn", me.LastName)
	assert.Equal(t, "ida@example.com", me.Email)
	assert.Equal(t, []string{roles.RoleReader}, me.Roles)

	rec = httptest.NewRecorder()
	h.UpdateProfile(rec, req.WithContext(context.Background()))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader([]byte(`{"email":"keine-mail"}`)))
	bad.Header.Set("Content-Type", "application/json")
	bad = bad.WithContext(auth.NewContext(bad.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.UpdateProfile(rec, bad)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMeAndChangePassword(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := usersqlite.New(st.DB())
	require.NoError(t, err)
	svc := identity.New(repo, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16), roles.NewService())
	user, plain, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "reader", Roles: []string{roles.RoleReader}})
	require.NoError(t, err)
	h := &meHandler{ident: svc}

	rec := httptest.NewRecorder()
	h.GetMe(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	unknown := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	unknown = unknown.WithContext(auth.NewContext(unknown.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": "missing"}}, nil, false))
	h.GetMe(rec, unknown)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.GetMe(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var me MeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.Equal(t, "reader", me.Username)

	rec = httptest.NewRecorder()
	change := httptest.NewRequest(http.MethodPost, "/api/v1/me/password", bytes.NewReader([]byte(`{"oldPassword":"wrong","newPassword":"new-secret"}`)))
	change.Header.Set("Content-Type", "application/json")
	change = change.WithContext(auth.NewContext(change.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.ChangePassword(rec, change)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	change = httptest.NewRequest(http.MethodPost, "/api/v1/me/password", bytes.NewReader([]byte(`{"oldPassword":"`+plain+`","newPassword":"short"}`)))
	change.Header.Set("Content-Type", "application/json")
	change = change.WithContext(auth.NewContext(change.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.ChangePassword(rec, change)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	same := `{"oldPassword":"` + plain + `","newPassword":"` + plain + `"}`
	change = httptest.NewRequest(http.MethodPost, "/api/v1/me/password", bytes.NewReader([]byte(same)))
	change.Header.Set("Content-Type", "application/json")
	change = change.WithContext(auth.NewContext(change.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.ChangePassword(rec, change)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	change = httptest.NewRequest(http.MethodPost, "/api/v1/me/password", bytes.NewReader([]byte(`{`)))
	change.Header.Set("Content-Type", "application/json")
	change = change.WithContext(auth.NewContext(change.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.ChangePassword(rec, change)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	change = httptest.NewRequest(http.MethodPost, "/api/v1/me/password", bytes.NewReader([]byte(`{"oldPassword":"`+plain+`","newPassword":"new-secret"}`)))
	change.Header.Set("Content-Type", "application/json")
	change = change.WithContext(auth.NewContext(change.Context(), &auth.JWT{IsValid: true, Payload: map[string]any{"sub": user.ID}}, nil, false))
	h.ChangePassword(rec, change)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
