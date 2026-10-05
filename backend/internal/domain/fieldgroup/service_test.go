package fieldgroup

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRejectsDuplicateFieldNames(t *testing.T) {
	svc := New(newMemStore())
	_, err := svc.Create(context.Background(), Input{
		Name: "beleg",
		Fields: []Field{
			{Name: "Amount", ValueType: ValueDecimal},
			{Name: "amount", ValueType: ValueText},
		},
	})
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestCreateAndRename(t *testing.T) {
	svc := New(newMemStore())
	svc.ids = func() string { return "fg1" }
	created, err := svc.Create(context.Background(), Input{
		Name:   " beleg ",
		Labels: Text{De: " Beleg ", En: "Voucher"},
		Fields: []Field{{
			Name:        "number",
			Labels:      Text{De: "Nummer", En: "Number"},
			Description: Text{De: "Belegnummer", En: "Voucher number"},
			ValueType:   ValueText,
			Mandatory:   true,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "fg1", created.ID)
	assert.Equal(t, "beleg", created.Name)
	assert.Equal(t, "Beleg", created.Labels.De)
	assert.Equal(t, ValueText, created.Fields[0].ValueType)
	assert.True(t, created.Fields[0].Mandatory)

	_, err = svc.Create(context.Background(), Input{Name: "BELEG"})
	assert.ErrorIs(t, err, ErrAlreadyExists)

	updated, err := svc.Update(context.Background(), created.ID, Input{
		Name:   "parties",
		Fields: []Field{{Name: "when", ValueType: ValueDateTime}},
	})
	require.NoError(t, err)
	assert.Equal(t, "parties", updated.Name)
	require.Len(t, updated.Fields, 1)
	assert.Equal(t, ValueDateTime, updated.Fields[0].ValueType)
}

func TestUpdateKeepsOwnName(t *testing.T) {
	svc := New(newMemStore())
	created, err := svc.Create(context.Background(), Input{Name: "beleg"})
	require.NoError(t, err)
	updated, err := svc.Update(context.Background(), created.ID, Input{
		Name: "beleg",
		Fields: []Field{
			{Name: "flag", ValueType: ValueBool},
			{Name: "count", ValueType: ValueInt},
			{Name: "rate", ValueType: ValueDouble},
			{Name: "note", ValueType: ValueMultiline},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Len(t, updated.Fields, 4)
}

func TestCreateAcceptsIntrefAndFile(t *testing.T) {
	svc := New(newMemStore())
	created, err := svc.Create(context.Background(), Input{
		Name: "refs",
		Fields: []Field{
			{Name: "owner", ValueType: ValueIntRef},
			{Name: "attachment", ValueType: ValueFile},
		},
	})
	require.NoError(t, err)
	require.Len(t, created.Fields, 2)
	assert.Equal(t, ValueIntRef, created.Fields[0].ValueType)
	assert.Equal(t, ValueFile, created.Fields[1].ValueType)
}

func TestRejectsUnknownValueTypeAndBadName(t *testing.T) {
	svc := New(newMemStore())
	_, err := svc.Create(context.Background(), Input{
		Name:   "beleg",
		Fields: []Field{{Name: "amount", ValueType: "money"}},
	})
	assert.ErrorIs(t, err, ErrInvalid)

	_, err = svc.Create(context.Background(), Input{Name: "1beleg"})
	assert.ErrorIs(t, err, ErrInvalid)

	_, err = svc.Create(context.Background(), Input{Name: strings.Repeat("a", maxNameLen+1)})
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestEnsureBuiltinImportsMissingGroups(t *testing.T) {
	st := newMemStore()
	svc := New(st)
	require.NoError(t, svc.EnsureBuiltin(context.Background()))
	system, err := st.GetByID(context.Background(), "system")
	require.NoError(t, err)
	assert.True(t, system.Readonly)
	assert.Equal(t, "system", system.Name)
	require.NotEmpty(t, system.Fields)
	var creator Field
	for _, field := range system.Fields {
		if field.Name == "creator" {
			creator = field
		}
	}
	assert.Equal(t, "intref:user", creator.ValueType)

	before, err := st.List(context.Background())
	require.NoError(t, err)
	require.NoError(t, svc.EnsureBuiltin(context.Background()))
	after, err := st.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, after, len(before))
}

func TestReadonlyGroupRejectsChange(t *testing.T) {
	st := newMemStore()
	require.NoError(t, st.Create(context.Background(), Group{ID: "system", Name: "system", Readonly: true}))
	svc := New(st)
	_, err := svc.Update(context.Background(), "system", Input{Name: "other"})
	assert.ErrorIs(t, err, ErrReadonly)
	assert.ErrorIs(t, svc.Delete(context.Background(), "system"), ErrReadonly)
	stored, err := st.GetByID(context.Background(), "system")
	require.NoError(t, err)
	assert.Equal(t, "system", stored.Name)
}

func TestDelete(t *testing.T) {
	svc := New(newMemStore())
	created, err := svc.Create(context.Background(), Input{Name: "beleg"})
	require.NoError(t, err)
	require.NoError(t, svc.Delete(context.Background(), created.ID))
	_, err = svc.Get(context.Background(), created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, svc.Delete(context.Background(), created.ID), ErrNotFound)
}

type memStore struct {
	groups map[string]Group
}

func newMemStore() *memStore {
	return &memStore{groups: map[string]Group{}}
}

func (m *memStore) List(context.Context) ([]Group, error) {
	out := make([]Group, 0, len(m.groups))
	for _, group := range m.groups {
		out = append(out, group)
	}
	return out, nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*Group, error) {
	group, ok := m.groups[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &group, nil
}

func (m *memStore) GetByName(_ context.Context, name string) (*Group, error) {
	for _, group := range m.groups {
		if strings.EqualFold(group.Name, name) {
			cp := group
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memStore) Create(_ context.Context, group Group) error {
	if _, err := m.GetByName(context.Background(), group.Name); err == nil {
		return ErrAlreadyExists
	}
	m.groups[group.ID] = group
	return nil
}

func (m *memStore) Update(_ context.Context, group Group) error {
	if _, ok := m.groups[group.ID]; !ok {
		return ErrNotFound
	}
	m.groups[group.ID] = group
	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	if _, ok := m.groups[id]; !ok {
		return ErrNotFound
	}
	delete(m.groups, id)
	return nil
}
