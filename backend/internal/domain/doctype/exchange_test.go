package doctype

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

func TestExportOmitsReadonlyGroups(t *testing.T) {
	svc, groups := exchangeService(t)
	_, err := groups.Create(context.Background(), fieldgroup.Input{
		Name:   "parties",
		Labels: fieldgroup.Text{De: "Parteien"},
		Fields: []fieldgroup.Field{{Name: "title", ValueType: fieldgroup.ValueText, Mandatory: true}},
	})
	require.NoError(t, err)
	party, err := groupByName(groups, "parties")
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), Input{
		Name:        "invoice",
		Labels:      Text{De: "Rechnung"},
		FieldGroups: []string{fieldgroup.SystemGroupID, party.ID},
	})
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), Input{
		Name:        "contract",
		FieldGroups: []string{fieldgroup.SystemGroupID, party.ID},
	})
	require.NoError(t, err)

	ex, err := svc.Export(context.Background())
	require.NoError(t, err)
	require.Len(t, ex.DocumentTypes, 2)
	assert.Equal(t, "contract", ex.DocumentTypes[0].Name)
	assert.NotEmpty(t, ex.DocumentTypes[1].ID)
	assert.Equal(t, []string{fieldgroup.SystemGroupID, party.ID}, ex.DocumentTypes[1].FieldGroups)
	require.Len(t, ex.FieldGroups, 1)
	assert.Equal(t, party.ID, ex.FieldGroups[0].ID)
	assert.Equal(t, "parties", ex.FieldGroups[0].Name)
	assert.Equal(t, "title", ex.FieldGroups[0].Fields[0].Name)
	assert.True(t, ex.FieldGroups[0].Fields[0].Mandatory)
}

func TestImportMatchesIDAndAsksOnConflict(t *testing.T) {
	svc, groups := exchangeService(t)
	system, err := groups.Get(context.Background(), fieldgroup.SystemGroupID)
	require.NoError(t, err)
	party, err := groups.CreateWithID(context.Background(), "fg-parties", fieldgroup.Input{
		Name: "parties",
		Fields: []fieldgroup.Field{{
			Name:      "title",
			ValueType: fieldgroup.ValueText,
		}},
	})
	require.NoError(t, err)
	created, err := svc.createAs(context.Background(), "type-invoice", Input{
		Name:        "invoice",
		Labels:      Text{De: "Rechnung"},
		FieldGroups: []string{fieldgroup.SystemGroupID, party.ID},
	})
	require.NoError(t, err)

	same := Exchange{
		DocumentTypes: []ExchangeType{{
			ID:          created.ID,
			Name:        "invoice",
			Labels:      Text{De: "Rechnung"},
			FieldGroups: []string{fieldgroup.SystemGroupID, party.ID},
		}},
		FieldGroups: []ExchangeGroup{{
			ID:   party.ID,
			Name: "parties",
			Fields: []ExchangeField{{
				Name:      "title",
				ValueType: fieldgroup.ValueText,
			}},
		}},
	}
	conflicts, err := svc.Preview(context.Background(), same)
	require.NoError(t, err)
	assert.Empty(t, conflicts)

	changed := same
	changed.DocumentTypes = []ExchangeType{{
		ID:          created.ID,
		Name:        "invoice",
		Labels:      Text{De: "Beleg"},
		FieldGroups: []string{fieldgroup.SystemGroupID, party.ID},
	}}
	changed.FieldGroups = append([]ExchangeGroup(nil), same.FieldGroups...)
	changed.FieldGroups = append(changed.FieldGroups, ExchangeGroup{
		ID:   fieldgroup.SystemGroupID,
		Name: "system",
		Fields: []ExchangeField{{
			Name:      "smuggled",
			ValueType: fieldgroup.ValueText,
		}},
	})
	conflicts, err = svc.Preview(context.Background(), changed)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, kindDocumentType, conflicts[0].Kind)
	assert.Equal(t, reasonSameID, conflicts[0].Reason)
	assert.Equal(t, "labelDe", conflicts[0].Changes[0].Field)
	assert.Equal(t, "Rechnung", conflicts[0].Changes[0].Before)
	assert.Equal(t, "Beleg", conflicts[0].Changes[0].After)

	_, err = svc.Import(context.Background(), changed, nil)
	assert.ErrorIs(t, err, ErrConflict)
	result, err := svc.Import(context.Background(), changed, []Decision{{
		Kind:   kindDocumentType,
		ID:     created.ID,
		Action: actionOverwrite,
	}})
	require.NoError(t, err)
	assert.Equal(t, ImportResult{DocumentTypes: 1}, result)
	unchanged, err := groups.Get(context.Background(), fieldgroup.SystemGroupID)
	require.NoError(t, err)
	assert.Equal(t, system.Fields, unchanged.Fields)
	listed, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Beleg", listed[0].Labels.De)

	clash := Exchange{
		DocumentTypes: []ExchangeType{{
			ID:          "type-other",
			Name:        "invoice",
			Labels:      Text{De: "Andere"},
			FieldGroups: []string{fieldgroup.SystemGroupID, "fg-other"},
		}},
		FieldGroups: []ExchangeGroup{{
			ID:   "fg-other",
			Name: "parties",
			Fields: []ExchangeField{{
				Name:      "note",
				ValueType: fieldgroup.ValueMultiline,
			}},
		}},
	}
	conflicts, err = svc.Preview(context.Background(), clash)
	require.NoError(t, err)
	require.Len(t, conflicts, 2)
	assert.Equal(t, reasonName, conflicts[0].Reason)
	assert.Equal(t, "parties (1)", conflicts[0].SuggestedName)
	assert.Equal(t, "invoice (1)", conflicts[1].SuggestedName)

	result, err = svc.Import(context.Background(), clash, []Decision{
		{Kind: kindFieldGroup, ID: "fg-other", Action: actionRename, Name: conflicts[0].SuggestedName},
		{Kind: kindDocumentType, ID: "type-other", Action: actionRename, Name: conflicts[1].SuggestedName},
	})
	require.NoError(t, err)
	assert.Equal(t, ImportResult{DocumentTypes: 1, FieldGroups: 1}, result)
	renamed, err := groups.Get(context.Background(), "fg-other")
	require.NoError(t, err)
	assert.Equal(t, "parties (1)", renamed.Name)
	kept, err := groups.Get(context.Background(), party.ID)
	require.NoError(t, err)
	assert.Equal(t, "parties", kept.Name)
}

func exchangeService(t *testing.T) (*Service, *fieldgroup.Service) {
	t.Helper()
	store := &groupStore{groups: map[string]fieldgroup.Group{}}
	require.NoError(t, store.Create(context.Background(), fieldgroup.Group{
		ID:       fieldgroup.SystemGroupID,
		Name:     "system",
		Readonly: true,
		Fields:   []fieldgroup.Field{{Name: "ID", ValueType: "string", Mandatory: true}},
	}))
	groups := fieldgroup.New(store)
	return New(newMemStore(), groups), groups
}

func groupByName(groups *fieldgroup.Service, name string) (fieldgroup.Group, error) {
	listed, err := groups.List(context.Background())
	if err != nil {
		return fieldgroup.Group{}, err
	}
	for _, group := range listed {
		if strings.EqualFold(group.Name, name) {
			return group, nil
		}
	}
	return fieldgroup.Group{}, fieldgroup.ErrNotFound
}

type groupStore struct {
	groups map[string]fieldgroup.Group
}

func (s *groupStore) List(context.Context) ([]fieldgroup.Group, error) {
	out := make([]fieldgroup.Group, 0, len(s.groups))
	for _, group := range s.groups {
		out = append(out, group)
	}
	return out, nil
}

func (s *groupStore) GetByID(_ context.Context, id string) (*fieldgroup.Group, error) {
	group, ok := s.groups[id]
	if !ok {
		return nil, fieldgroup.ErrNotFound
	}
	cp := group
	return &cp, nil
}

func (s *groupStore) GetByName(_ context.Context, name string) (*fieldgroup.Group, error) {
	for _, group := range s.groups {
		if strings.EqualFold(group.Name, name) {
			cp := group
			return &cp, nil
		}
	}
	return nil, fieldgroup.ErrNotFound
}

func (s *groupStore) Create(_ context.Context, group fieldgroup.Group) error {
	if _, err := s.GetByName(context.Background(), group.Name); err == nil {
		return fieldgroup.ErrAlreadyExists
	}
	s.groups[group.ID] = group
	return nil
}

func (s *groupStore) Update(_ context.Context, group fieldgroup.Group) error {
	if _, ok := s.groups[group.ID]; !ok {
		return fieldgroup.ErrNotFound
	}
	s.groups[group.ID] = group
	return nil
}

func (s *groupStore) Delete(_ context.Context, id string) error {
	if _, ok := s.groups[id]; !ok {
		return fieldgroup.ErrNotFound
	}
	delete(s.groups, id)
	return nil
}
