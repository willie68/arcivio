package doctypes

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
	"github.com/willie68/arcivio/internal/domain/doctype"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/roles"
)

func TestUpdateRejectsRemovingSystemGroup(t *testing.T) {
	svc := doctype.New(newMemStore(), knownGroups(fieldgroup.SystemGroupID, "parties"))
	h := &Handler{types: svc, roles: stubRoles{}}

	body := []byte(`{
		"name": "invoice",
		"labels": {"de": "Rechnung", "en": "Invoice"},
		"description": {"de": "", "en": ""},
		"fieldGroups": ["system", "parties"]
	}`)
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodPost, "/api/v1/document-types", bytes.NewReader(body)), roles.RoleAdmin)
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created TypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, []string{"system", "parties"}, created.FieldGroups)

	update := []byte(`{
		"name": "invoice",
		"labels": {"de": "Rechnung", "en": "Invoice"},
		"description": {"de": "", "en": ""},
		"fieldGroups": ["parties"]
	}`)
	rec = httptest.NewRecorder()
	put := asUser(httptest.NewRequest(http.MethodPut, "/api/v1/document-types/"+created.ID, bytes.NewReader(update)), roles.RoleAdmin)
	put.Header.Set("Content-Type", "application/json")
	put = withURLParam(put, "id", created.ID)
	h.Update(rec, put)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
	assert.Equal(t, "system-group", problem["key"])
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

type memStore struct {
	types map[string]doctype.Type
}

func newMemStore() *memStore {
	return &memStore{types: map[string]doctype.Type{}}
}

func (m *memStore) List(context.Context) ([]doctype.Type, error) {
	out := make([]doctype.Type, 0, len(m.types))
	for _, docType := range m.types {
		out = append(out, docType)
	}
	return out, nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*doctype.Type, error) {
	docType, ok := m.types[id]
	if !ok {
		return nil, doctype.ErrNotFound
	}
	cp := docType
	return &cp, nil
}

func (m *memStore) GetByName(_ context.Context, name string) (*doctype.Type, error) {
	for _, docType := range m.types {
		if docType.Name == name {
			cp := docType
			return &cp, nil
		}
	}
	return nil, doctype.ErrNotFound
}

func (m *memStore) Create(_ context.Context, docType doctype.Type) error {
	m.types[docType.ID] = docType
	return nil
}

func (m *memStore) Update(_ context.Context, docType doctype.Type) error {
	if _, ok := m.types[docType.ID]; !ok {
		return doctype.ErrNotFound
	}
	m.types[docType.ID] = docType
	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	if _, ok := m.types[id]; !ok {
		return doctype.ErrNotFound
	}
	delete(m.types, id)
	return nil
}

type groupCatalog map[string]fieldgroup.Group

func knownGroups(ids ...string) groupCatalog {
	catalog := groupCatalog{}
	for _, id := range ids {
		catalog[id] = fieldgroup.Group{ID: id, Name: id}
	}
	return catalog
}

func (g groupCatalog) List(context.Context) ([]fieldgroup.Group, error) {
	out := make([]fieldgroup.Group, 0, len(g))
	for _, group := range g {
		out = append(out, group)
	}
	return out, nil
}

func (g groupCatalog) Get(_ context.Context, id string) (*fieldgroup.Group, error) {
	group, ok := g[id]
	if !ok {
		return nil, fieldgroup.ErrNotFound
	}
	return &group, nil
}

func (g groupCatalog) Create(context.Context, fieldgroup.Input) (*fieldgroup.Group, error) {
	return nil, fieldgroup.ErrInvalid
}

func (g groupCatalog) CreateWithID(context.Context, string, fieldgroup.Input) (*fieldgroup.Group, error) {
	return nil, fieldgroup.ErrInvalid
}

func (g groupCatalog) Update(context.Context, string, fieldgroup.Input) (*fieldgroup.Group, error) {
	return nil, fieldgroup.ErrInvalid
}
