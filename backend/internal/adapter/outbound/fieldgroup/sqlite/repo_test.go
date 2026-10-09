package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/models"
)

func TestFieldGroupRepoCRUD(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "fg.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	group := fieldgroup.FieldGroup{
		ID:          "fg1",
		Name:        "beleg",
		Labels:      models.LabelText{De: "Beleg", En: "Voucher"},
		Description: models.LabelText{De: "Kopf", En: "Header"},
		Readonly:    true,
		Fields: []fieldgroup.Field{{
			Name:        "amount",
			Labels:      models.LabelText{De: "Betrag", En: "Amount"},
			Description: models.LabelText{De: "Brutto", En: "Gross"},
			ValueType:   models.ValueTypeDecimal,
			Mandatory:   true,
		}},
	}
	require.NoError(t, repo.Create(context.Background(), group))

	byName, err := repo.GetByName(context.Background(), "BELEG")
	require.NoError(t, err)
	assert.Equal(t, "fg1", byName.ID)
	assert.Equal(t, "Beleg", byName.Labels.De)
	require.Len(t, byName.Fields, 1)
	assert.Equal(t, models.ValueTypeDecimal, byName.Fields[0].ValueType)
	assert.True(t, byName.Fields[0].Mandatory)
	assert.True(t, byName.Readonly)
	assert.Equal(t, "Brutto", byName.Fields[0].Description.De)

	group.Name = "parties"
	group.Fields = []fieldgroup.Field{{Name: "at", ValueType: models.ValueTypeDateTime}}
	require.NoError(t, repo.Update(context.Background(), group))

	listed, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "parties", listed[0].Name)

	require.NoError(t, repo.Delete(context.Background(), "fg1"))
	_, err = repo.GetByID(context.Background(), "fg1")
	assert.ErrorIs(t, err, fieldgroup.ErrNotFound)
}

func TestFieldGroupRepoNameIsUniqueIgnoringCase(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "fg.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), fieldgroup.FieldGroup{ID: "a", Name: "beleg"}))
	err = repo.Create(context.Background(), fieldgroup.FieldGroup{ID: "b", Name: "Beleg"})
	assert.ErrorIs(t, err, fieldgroup.ErrAlreadyExists)
}
