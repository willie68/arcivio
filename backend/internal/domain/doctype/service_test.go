package doctype

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

func TestCreateRequiresSystemGroup(t *testing.T) {
	svc := New(newMemStore(), knownGroups(fieldgroup.SystemGroupID, "parties"))
	_, err := svc.Create(context.Background(), Input{
		Name:        "invoice",
		FieldGroups: []string{"parties"},
	})
	assert.ErrorIs(t, err, ErrSystemGroup)
}

func TestCreateKeepsGroupOrder(t *testing.T) {
	svc := New(newMemStore(), knownGroups(fieldgroup.SystemGroupID, "parties"))
	svc.ids = func() string { return "t1" }
	created, err := svc.Create(context.Background(), Input{
		Name:        " invoice ",
		Labels:      Text{De: " Rechnung ", En: "Invoice"},
		FieldGroups: []string{fieldgroup.SystemGroupID, "parties"},
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", created.ID)
	assert.Equal(t, "invoice", created.Name)
	assert.Equal(t, "Rechnung", created.Labels.De)
	assert.Equal(t, []string{fieldgroup.SystemGroupID, "parties"}, created.FieldGroups)
}

func TestUpdateRejectsRemovingSystemGroup(t *testing.T) {
	groups := knownGroups(fieldgroup.SystemGroupID, "parties")
	svc := New(newMemStore(), groups)
	created, err := svc.Create(context.Background(), Input{
		Name:        "invoice",
		FieldGroups: []string{fieldgroup.SystemGroupID, "parties"},
	})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), created.ID, Input{
		Name:        "invoice",
		FieldGroups: []string{"parties"},
	})
	assert.ErrorIs(t, err, ErrSystemGroup)

	stored, err := svc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{fieldgroup.SystemGroupID, "parties"}, stored.FieldGroups)
}

func TestUpdateRejectsUnknownGroup(t *testing.T) {
	svc := New(newMemStore(), knownGroups(fieldgroup.SystemGroupID))
	created, err := svc.Create(context.Background(), Input{
		Name:        "invoice",
		FieldGroups: []string{fieldgroup.SystemGroupID},
	})
	require.NoError(t, err)
	_, err = svc.Update(context.Background(), created.ID, Input{
		Name:        "invoice",
		FieldGroups: []string{fieldgroup.SystemGroupID, "missing"},
	})
	assert.ErrorIs(t, err, ErrUnknownGroup)
}

type memStore struct {
	types map[string]Type
}

func newMemStore() *memStore {
	return &memStore{types: map[string]Type{}}
}

func (m *memStore) List(context.Context) ([]Type, error) {
	out := make([]Type, 0, len(m.types))
	for _, docType := range m.types {
		out = append(out, docType)
	}
	return out, nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*Type, error) {
	docType, ok := m.types[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := docType
	return &cp, nil
}

func (m *memStore) GetByName(_ context.Context, name string) (*Type, error) {
	for _, docType := range m.types {
		if docType.Name == name {
			cp := docType
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memStore) Create(_ context.Context, docType Type) error {
	m.types[docType.ID] = docType
	return nil
}

func (m *memStore) Update(_ context.Context, docType Type) error {
	if _, ok := m.types[docType.ID]; !ok {
		return ErrNotFound
	}
	m.types[docType.ID] = docType
	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	if _, ok := m.types[id]; !ok {
		return ErrNotFound
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
