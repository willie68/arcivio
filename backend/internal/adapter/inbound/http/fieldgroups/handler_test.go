package fieldgroups

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/roles"
)

func TestListRequiresAdmin(t *testing.T) {
	h := &Handler{groups: &stubGroups{}, roles: stubRoles{}}

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/field-groups", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	h.List(rec, asUser(httptest.NewRequest(http.MethodGet, "/api/v1/field-groups", nil), roles.RoleClerk))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateListAndDelete(t *testing.T) {
	svc := fieldgroup.New(&memGroups{groups: map[string]fieldgroup.FieldGroup{}})
	h := &Handler{groups: svc, roles: stubRoles{}}

	body := []byte(`{
		"name": "beleg",
		"labels": {"de": "Beleg", "en": "Voucher"},
		"description": {"de": "", "en": ""},
		"fields": [{
			"name": "amount",
			"labels": {"de": "Betrag", "en": "Amount"},
			"description": {"de": "Brutto", "en": "Gross"},
			"valueType": "decimal",
			"mandatory": true
		}]
	}`)
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodPost, "/api/v1/field-groups", bytes.NewReader(body)), roles.RoleAdmin)
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created GroupResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "beleg", created.Name)
	assert.Equal(t, "decimal", created.Fields[0].ValueType)
	assert.True(t, created.Fields[0].Mandatory)
	assert.NotEmpty(t, created.ID)

	rec = httptest.NewRecorder()
	h.List(rec, asUser(httptest.NewRequest(http.MethodGet, "/api/v1/field-groups", nil), roles.RoleAdmin))
	require.Equal(t, http.StatusOK, rec.Code)
	var listed GroupListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Items, 1)

	rec = httptest.NewRecorder()
	del := asUser(httptest.NewRequest(http.MethodDelete, "/api/v1/field-groups/"+created.ID, nil), roles.RoleAdmin)
	del = withURLParam(del, "id", created.ID)
	h.Delete(rec, del)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func asUser(req *http.Request, roleNames ...string) *http.Request {
	claimed := make([]any, len(roleNames))
	for i, name := range roleNames {
		claimed[i] = name
	}
	return req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": "admin-id", "roles": claimed},
	}, nil, false))
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	route := chi.NewRouteContext()
	route.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
}

type stubRoles struct{}

func (stubRoles) HasRole(checkRole string, desiredRole string) bool {
	return checkRole == desiredRole
}

type stubGroups struct{}

func (stubGroups) List(context.Context) ([]fieldgroup.FieldGroup, error) { return nil, nil }
func (stubGroups) Get(context.Context, string) (*fieldgroup.FieldGroup, error) {
	return nil, fieldgroup.ErrNotFound
}
func (stubGroups) Create(context.Context, fieldgroup.Input) (*fieldgroup.FieldGroup, error) {
	return nil, fieldgroup.ErrInvalid
}
func (stubGroups) Update(context.Context, string, fieldgroup.Input) (*fieldgroup.FieldGroup, error) {
	return nil, fieldgroup.ErrNotFound
}
func (stubGroups) Delete(context.Context, string) error { return fieldgroup.ErrNotFound }

type memGroups struct {
	groups map[string]fieldgroup.FieldGroup
}

func (m *memGroups) List(context.Context) ([]fieldgroup.FieldGroup, error) {
	out := make([]fieldgroup.FieldGroup, 0, len(m.groups))
	for _, group := range m.groups {
		out = append(out, group)
	}
	return out, nil
}

func (m *memGroups) GetByID(_ context.Context, id string) (*fieldgroup.FieldGroup, error) {
	group, ok := m.groups[id]
	if !ok {
		return nil, fieldgroup.ErrNotFound
	}
	return &group, nil
}

func (m *memGroups) GetByName(_ context.Context, name string) (*fieldgroup.FieldGroup, error) {
	for _, group := range m.groups {
		if group.Name == name {
			cp := group
			return &cp, nil
		}
	}
	return nil, fieldgroup.ErrNotFound
}

func (m *memGroups) Create(_ context.Context, group fieldgroup.FieldGroup) error {
	m.groups[group.ID] = group
	return nil
}

func (m *memGroups) Update(_ context.Context, group fieldgroup.FieldGroup) error {
	m.groups[group.ID] = group
	return nil
}

func (m *memGroups) Delete(_ context.Context, id string) error {
	delete(m.groups, id)
	return nil
}
