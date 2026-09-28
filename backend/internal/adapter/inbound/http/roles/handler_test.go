package roles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/roles"
)

func TestListRolesRequiresAdmin(t *testing.T) {
	rs := newMockRolesService(t)
	h := &Handler{roles: rs}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	h.List(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rs.EXPECT().HasRole(roles.RoleClerk, roles.RoleAdmin).Return(false).Once()
	rec = httptest.NewRecorder()
	req = asUser(httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil), "clerk-id", roles.RoleClerk)
	h.List(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	admin := roles.RoleDefinition{
		Name:        roles.RoleAdmin,
		Labels:      roles.Text{De: "Administrator", En: "Administrator"},
		Description: roles.Text{De: "Verwaltet Benutzer.", En: "Manages users."},
	}
	listed := []roles.RoleDefinition{
		admin,
		{Name: roles.RoleArchivist, Labels: roles.Text{De: "Archivar", En: "Archivist"}},
		{Name: roles.RoleClerk, Labels: roles.Text{De: "Sachbearbeiter", En: "Clerk"}},
		{Name: roles.RoleReader, Labels: roles.Text{De: "Leser", En: "Reader"}},
	}
	rs.EXPECT().HasRole(roles.RoleAdmin, roles.RoleAdmin).Return(true).Once()
	rs.EXPECT().GetRoles().Return(listed).Once()

	rec = httptest.NewRecorder()
	req = asUser(httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil), "admin-id", roles.RoleAdmin)
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body RoleListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 4)
	assert.Equal(t, roles.RoleAdmin, body.Items[0].Name)
	assert.Equal(t, "Administrator", body.Items[0].Labels.De)
	assert.Equal(t, "Administrator", body.Items[0].Labels.En)
	assert.Equal(t, "Verwaltet Benutzer.", body.Items[0].Description.De)
	assert.Equal(t, "Manages users.", body.Items[0].Description.En)
	assert.Equal(t, roles.RoleReader, body.Items[3].Name)
}

func asUser(req *http.Request, userID string, roleNames ...string) *http.Request {
	claimed := make([]any, len(roleNames))
	for i, name := range roleNames {
		claimed[i] = name
	}
	return req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": userID, "roles": claimed},
	}, nil, false))
}
