package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/doctype"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

func TestDocumentTypeRepoCRUD(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "dt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	docType := doctype.Type{
		ID:          "t1",
		Name:        "invoice",
		Labels:      doctype.Text{De: "Rechnung", En: "Invoice"},
		Description: doctype.Text{De: "Beleg", En: "Voucher"},
		FieldGroups: []string{fieldgroup.SystemGroupID, "parties"},
	}
	require.NoError(t, repo.Create(context.Background(), docType))

	byName, err := repo.GetByName(context.Background(), "INVOICE")
	require.NoError(t, err)
	assert.Equal(t, "t1", byName.ID)
	assert.Equal(t, []string{fieldgroup.SystemGroupID, "parties"}, byName.FieldGroups)
	assert.Equal(t, "Rechnung", byName.Labels.De)

	docType.Name = "contract"
	docType.FieldGroups = []string{"parties", fieldgroup.SystemGroupID}
	require.NoError(t, repo.Update(context.Background(), docType))

	listed, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "contract", listed[0].Name)
	assert.Equal(t, []string{"parties", fieldgroup.SystemGroupID}, listed[0].FieldGroups)

	require.NoError(t, repo.Delete(context.Background(), "t1"))
	_, err = repo.GetByID(context.Background(), "t1")
	assert.ErrorIs(t, err, doctype.ErrNotFound)
}
