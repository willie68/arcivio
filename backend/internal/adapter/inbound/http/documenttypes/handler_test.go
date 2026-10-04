package documenttypes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/documenttype"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/roles"
)

func TestListRequiresAdmin(t *testing.T) {
	h := &Handler{types: &stubTypes{}, roles: stubRoles{}}

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/document-types", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	h.List(rec, asUser(httptest.NewRequest(http.MethodGet, "/api/v1/document-types", nil), roles.RoleClerk))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateListsFieldGroups(t *testing.T) {
	groups := &memGroups{ids: map[string]struct{}{"g1": {}}}
	store := &memTypes{items: map[string]documenttype.Type{}}
	h := &Handler{types: documenttype.New(store, groups), roles: stubRoles{}}

	body := []byte(`{
		"name": "invoice",
		"labels": {"de": "Rechnung", "en": "Invoice"},
		"description": {"de": "", "en": ""},
		"fieldGroupIds": ["g1"]
	}`)
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodPost, "/api/v1/document-types", bytes.NewReader(body)), roles.RoleAdmin)
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created TypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "invoice", created.Name)
	assert.Equal(t, []string{"g1"}, created.FieldGroupIDs)
	assert.NotEmpty(t, created.ID)
}

func TestGetUpdateAndDelete(t *testing.T) {
	groups := &memGroups{ids: map[string]struct{}{"g1": {}, "g2": {}}}
	store := &memTypes{items: map[string]documenttype.Type{}}
	h := &Handler{types: documenttype.New(store, groups), roles: stubRoles{}}

	body := []byte(`{"name":"invoice","fieldGroupIds":["g1"]}`)
	rec := httptest.NewRecorder()
	h.Create(rec, adminJSON(http.MethodPost, "/api/v1/document-types", body, ""))
	require.Equal(t, http.StatusCreated, rec.Code)
	var created TypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = httptest.NewRecorder()
	h.Get(rec, adminJSON(http.MethodGet, "/api/v1/document-types/"+created.ID, nil, created.ID))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	h.Update(rec, adminJSON(http.MethodPut, "/api/v1/document-types/"+created.ID, []byte(`{"name":"contract","fieldGroupIds":["g2","g1"]}`), created.ID))
	require.Equal(t, http.StatusOK, rec.Code)
	var updated TypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, []string{"g2", "g1"}, updated.FieldGroupIDs)

	rec = httptest.NewRecorder()
	h.Delete(rec, adminJSON(http.MethodDelete, "/api/v1/document-types/"+created.ID, nil, created.ID))
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = httptest.NewRecorder()
	h.Get(rec, adminJSON(http.MethodGet, "/api/v1/document-types/"+created.ID, nil, created.ID))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTypeErrorMapping(t *testing.T) {
	h := &Handler{types: stubTypes{}, roles: stubRoles{}}

	rec := httptest.NewRecorder()
	h.Create(rec, adminJSON(http.MethodPost, "/api/v1/document-types", []byte(`{`), ""))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	h.Create(rec, adminJSON(http.MethodPost, "/api/v1/document-types", []byte(`{"name":"invoice"}`), ""))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	h.Get(rec, adminJSON(http.MethodGet, "/api/v1/document-types/missing", nil, "missing"))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	h.Update(rec, adminJSON(http.MethodPut, "/api/v1/document-types/missing", []byte(`{"name":"invoice"}`), "missing"))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	h.List(rec, adminJSON(http.MethodGet, "/api/v1/document-types", nil, ""))
	assert.Equal(t, http.StatusOK, rec.Code)

	h = &Handler{types: errTypes{}, roles: stubRoles{}}
	rec = httptest.NewRecorder()
	h.List(rec, adminJSON(http.MethodGet, "/api/v1/document-types", nil, ""))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	rec = httptest.NewRecorder()
	h.Delete(rec, adminJSON(http.MethodDelete, "/api/v1/document-types/x", nil, "x"))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	groups := &memGroups{ids: map[string]struct{}{}}
	store := &memTypes{items: map[string]documenttype.Type{}}
	h = &Handler{types: documenttype.New(store, groups), roles: stubRoles{}}
	rec = httptest.NewRecorder()
	h.Create(rec, adminJSON(http.MethodPost, "/api/v1/document-types", []byte(`{"name":"invoice"}`), ""))
	require.Equal(t, http.StatusCreated, rec.Code)
	rec = httptest.NewRecorder()
	h.Create(rec, adminJSON(http.MethodPost, "/api/v1/document-types", []byte(`{"name":"invoice"}`), ""))
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestAdminRoleAsStringSlice(t *testing.T) {
	h := &Handler{types: &stubTypes{}, roles: stubRoles{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-types", nil)
	req = req.WithContext(auth.NewContext(req.Context(), &auth.JWT{
		IsValid: true,
		Payload: map[string]any{"sub": "admin-id", "roles": []string{roles.RoleAdmin}},
	}, nil, false))
	rec := httptest.NewRecorder()
	h.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	got := toResponse(documenttype.Type{Name: "invoice"})
	assert.Empty(t, got.FieldGroupIDs)
	assert.NotNil(t, got.FieldGroupIDs)
}

func adminJSON(method, target string, body []byte, id string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asUser(req, roles.RoleAdmin)
	if id != "" {
		req = withURLParam(req, "id", id)
	}
	return req
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	route := chi.NewRouteContext()
	route.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
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

type stubRoles struct{}

func (stubRoles) HasRole(checkRole string, desiredRole string) bool {
	return checkRole == desiredRole
}

type stubTypes struct{}

func (stubTypes) List(context.Context) ([]documenttype.Type, error) { return nil, nil }
func (stubTypes) Get(context.Context, string) (*documenttype.Type, error) {
	return nil, documenttype.ErrNotFound
}
func (stubTypes) Create(context.Context, documenttype.Input) (*documenttype.Type, error) {
	return nil, documenttype.ErrInvalid
}
func (stubTypes) Update(context.Context, string, documenttype.Input) (*documenttype.Type, error) {
	return nil, documenttype.ErrNotFound
}
func (stubTypes) Delete(context.Context, string) error { return documenttype.ErrNotFound }

type errTypes struct{}

func (errTypes) List(context.Context) ([]documenttype.Type, error) {
	return nil, errors.New("db down")
}
func (errTypes) Get(context.Context, string) (*documenttype.Type, error) {
	return nil, errors.New("db down")
}
func (errTypes) Create(context.Context, documenttype.Input) (*documenttype.Type, error) {
	return nil, errors.New("db down")
}
func (errTypes) Update(context.Context, string, documenttype.Input) (*documenttype.Type, error) {
	return nil, errors.New("db down")
}
func (errTypes) Delete(context.Context, string) error { return errors.New("db down") }

type memGroups struct {
	ids map[string]struct{}
}

func (m *memGroups) GetByID(_ context.Context, id string) (*fieldgroup.Group, error) {
	if _, ok := m.ids[id]; !ok {
		return nil, fieldgroup.ErrNotFound
	}
	return &fieldgroup.Group{ID: id}, nil
}

type memTypes struct {
	items map[string]documenttype.Type
}

func (m *memTypes) List(context.Context) ([]documenttype.Type, error) {
	out := make([]documenttype.Type, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

func (m *memTypes) GetByID(_ context.Context, id string) (*documenttype.Type, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, documenttype.ErrNotFound
	}
	return &item, nil
}

func (m *memTypes) GetByName(_ context.Context, name string) (*documenttype.Type, error) {
	for _, item := range m.items {
		if item.Name == name {
			cp := item
			return &cp, nil
		}
	}
	return nil, documenttype.ErrNotFound
}

func (m *memTypes) Create(_ context.Context, docType documenttype.Type) error {
	m.items[docType.ID] = docType
	return nil
}

func (m *memTypes) Update(_ context.Context, docType documenttype.Type) error {
	m.items[docType.ID] = docType
	return nil
}

func (m *memTypes) Delete(_ context.Context, id string) error {
	delete(m.items, id)
	return nil
}

func (m *memTypes) UsesGroup(context.Context, string) (bool, error) { return false, nil }
