package documenttype

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

func TestCreateKeepsFieldGroupOrder(t *testing.T) {
	groups := &memGroups{ids: map[string]struct{}{"g1": {}, "g2": {}}}
	svc := New(newMemStore(), groups)
	svc.ids = func() string { return "t1" }

	created, err := svc.Create(context.Background(), Input{
		Name:          " invoice ",
		Labels:        Text{De: " Rechnung ", En: "Invoice"},
		Description:   Text{De: "Beleg", En: "Voucher"},
		FieldGroupIDs: []string{"g2", "g1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", created.ID)
	assert.Equal(t, "invoice", created.Name)
	assert.Equal(t, "Rechnung", created.Labels.De)
	assert.Equal(t, []string{"g2", "g1"}, created.FieldGroupIDs)

	_, err = svc.Create(context.Background(), Input{Name: "INVOICE"})
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestRejectsUnknownOrRepeatedGroup(t *testing.T) {
	svc := New(newMemStore(), &memGroups{ids: map[string]struct{}{"g1": {}}})
	_, err := svc.Create(context.Background(), Input{Name: "invoice", FieldGroupIDs: []string{"missing"}})
	assert.ErrorIs(t, err, ErrInvalid)

	_, err = svc.Create(context.Background(), Input{Name: "invoice", FieldGroupIDs: []string{"g1", "g1"}})
	assert.ErrorIs(t, err, ErrInvalid)

	_, err = svc.Create(context.Background(), Input{Name: "1invoice"})
	assert.ErrorIs(t, err, ErrInvalid)
	_, err = svc.Create(context.Background(), Input{Name: strings.Repeat("a", maxNameLen+1)})
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestListAndDelete(t *testing.T) {
	svc := New(&memStore{items: map[string]Type{}, listNil: true}, &memGroups{ids: map[string]struct{}{}})
	listed, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, listed)

	created, err := svc.Create(context.Background(), Input{Name: "invoice"})
	require.NoError(t, err)
	listed, err = svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)

	_, err = svc.Get(context.Background(), "  ")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, svc.Delete(context.Background(), "missing"), ErrNotFound)
	require.NoError(t, svc.Delete(context.Background(), created.ID))
	_, err = svc.Get(context.Background(), created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateRejectsTakenNameAndBlankGroup(t *testing.T) {
	groups := &memGroups{ids: map[string]struct{}{"g1": {}}, failID: "down"}
	svc := New(newMemStore(), groups)
	first, err := svc.Create(context.Background(), Input{Name: "invoice"})
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), Input{Name: "contract"})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), first.ID, Input{Name: "contract"})
	assert.ErrorIs(t, err, ErrAlreadyExists)
	_, err = svc.Update(context.Background(), "missing", Input{Name: "other"})
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = svc.Create(context.Background(), Input{Name: "note", FieldGroupIDs: []string{"  "}})
	assert.ErrorIs(t, err, ErrInvalid)
	_, err = svc.Create(context.Background(), Input{Name: "note", FieldGroupIDs: []string{"down"}})
	assert.EqualError(t, err, "lookup failed")
}

func TestUpdateKeepsOwnName(t *testing.T) {
	groups := &memGroups{ids: map[string]struct{}{"g1": {}}}
	svc := New(newMemStore(), groups)
	created, err := svc.Create(context.Background(), Input{Name: "invoice", FieldGroupIDs: []string{"g1"}})
	require.NoError(t, err)
	updated, err := svc.Update(context.Background(), created.ID, Input{Name: "invoice"})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Empty(t, updated.FieldGroupIDs)
}

type memGroups struct {
	ids    map[string]struct{}
	failID string
}

func (m *memGroups) GetByID(_ context.Context, id string) (*fieldgroup.Group, error) {
	if id == m.failID {
		return nil, errors.New("lookup failed")
	}
	if _, ok := m.ids[id]; !ok {
		return nil, fieldgroup.ErrNotFound
	}
	return &fieldgroup.Group{ID: id, Name: id}, nil
}

type memStore struct {
	items   map[string]Type
	listNil bool
}

func newMemStore() *memStore {
	return &memStore{items: map[string]Type{}}
}

func (m *memStore) List(context.Context) ([]Type, error) {
	if m.listNil {
		m.listNil = false
		return nil, nil
	}
	out := make([]Type, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*Type, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (m *memStore) GetByName(_ context.Context, name string) (*Type, error) {
	for _, item := range m.items {
		if strings.EqualFold(item.Name, name) {
			cp := item
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memStore) Create(_ context.Context, docType Type) error {
	m.items[docType.ID] = docType
	return nil
}

func (m *memStore) Update(_ context.Context, docType Type) error {
	if _, ok := m.items[docType.ID]; !ok {
		return ErrNotFound
	}
	m.items[docType.ID] = docType
	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	if _, ok := m.items[id]; !ok {
		return ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *memStore) UsesGroup(_ context.Context, groupID string) (bool, error) {
	for _, item := range m.items {
		for _, id := range item.FieldGroupIDs {
			if id == groupID {
				return true, nil
			}
		}
	}
	return false, nil
}
