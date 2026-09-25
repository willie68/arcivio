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
)

func TestUpdateOwnProfile(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := usersqlite.New(st.DB())
	require.NoError(t, err)
	svc := identity.New(repo, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16))
	user, _, err := svc.CreateUser(context.Background(), identity.NewUser{Username: "reader", Roles: []string{identity.RoleReader}})
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
	assert.Equal(t, []string{identity.RoleReader}, me.Roles)
}
