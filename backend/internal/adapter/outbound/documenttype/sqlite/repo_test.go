package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/documenttype"
)

func TestDocumentTypeRepoCRUD(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "dt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	item := documenttype.Type{
		ID:            "t1",
		Name:          "invoice",
		Labels:        documenttype.Text{De: "Rechnung", En: "Invoice"},
		Description:   documenttype.Text{De: "Eingang", En: "Incoming"},
		FieldGroupIDs: []string{"g2", "g1"},
	}
	require.NoError(t, repo.Create(context.Background(), item))

	byName, err := repo.GetByName(context.Background(), "INVOICE")
	require.NoError(t, err)
	assert.Equal(t, "t1", byName.ID)
	assert.Equal(t, []string{"g2", "g1"}, byName.FieldGroupIDs)
	assert.Equal(t, "Eingang", byName.Description.De)

	used, err := repo.UsesGroup(context.Background(), "g1")
	require.NoError(t, err)
	assert.True(t, used)

	item.FieldGroupIDs = nil
	item.Name = "contract"
	require.NoError(t, repo.Update(context.Background(), item))
	used, err = repo.UsesGroup(context.Background(), "g1")
	require.NoError(t, err)
	assert.False(t, used)

	byID, err := repo.GetByID(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "contract", byID.Name)

	err = repo.Update(context.Background(), documenttype.Type{ID: "missing", Name: "other"})
	assert.ErrorIs(t, err, documenttype.ErrNotFound)
	err = repo.Delete(context.Background(), "missing")
	assert.ErrorIs(t, err, documenttype.ErrNotFound)

	require.NoError(t, repo.Delete(context.Background(), "t1"))
	_, err = repo.GetByID(context.Background(), "t1")
	assert.ErrorIs(t, err, documenttype.ErrNotFound)
}

func TestDocumentTypeRepoMapsUniqueAndBadJSON(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "dt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	repo, err := New(st.DB())
	require.NoError(t, err)

	require.NoError(t, repo.Create(context.Background(), documenttype.Type{ID: "a", Name: "invoice"}))
	err = repo.Create(context.Background(), documenttype.Type{ID: "b", Name: "Invoice"})
	assert.ErrorIs(t, err, documenttype.ErrAlreadyExists)

	_, err = repo.db.Exec(`INSERT INTO document_types (id, name, definition, created_at, updated_at) VALUES ('c', 'broken', 'not-json', 't', 't')`)
	require.NoError(t, err)
	_, err = repo.GetByID(context.Background(), "c")
	assert.Error(t, err)
}
