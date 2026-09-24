package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
	"github.com/willie68/arcivio/internal/domain/identity"
)

func TestUserRepoCRUD(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	n, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	now := time.Now().UTC().Truncate(time.Second)
	u := identity.User{
		ID:                 "u1",
		Username:           "admin",
		PasswordHash:       "$argon2id$v=19$m=16,t=1,p=1$YWFhYWFhYWFhYWFhYWFhYQ$YmJiYmJiYmJiYmJiYmJiYg",
		Roles:              []string{identity.RoleAdmin},
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, repo.Create(context.Background(), u))

	got, err := repo.GetByUsername(context.Background(), "ADMIN")
	require.NoError(t, err)
	assert.Equal(t, "u1", got.ID)
	assert.True(t, strings.HasPrefix(got.PasswordHash, "$argon2id$"))
	assert.NotEqual(t, "admin", got.PasswordHash)
	assert.True(t, got.MustChangePassword)

	got.MustChangePassword = false
	got.PasswordHash = "$argon2id$v=19$m=16,t=1,p=1$changedchangedchang$changedchangedchangedch"
	got.UpdatedAt = now.Add(time.Minute)
	require.NoError(t, repo.Update(context.Background(), *got))

	byID, err := repo.GetByID(context.Background(), "u1")
	require.NoError(t, err)
	assert.False(t, byID.MustChangePassword)

	_, err = repo.GetByUsername(context.Background(), "missing")
	assert.ErrorIs(t, err, identity.ErrUserNotFound)
}

func TestUserRepoListOrdersByUsername(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	for _, name := range []string{"zeta", "alpha"} {
		require.NoError(t, repo.Create(context.Background(), identity.User{
			ID:           name,
			Username:     name,
			PasswordHash: "hash",
			Roles:        []string{identity.RoleReader},
			CreatedAt:    now,
			UpdatedAt:    now,
		}))
	}
	page, total, err := repo.List(context.Background(), 0, 1, identity.SortUsername, false, "")
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, page, 1)
	assert.Equal(t, "alpha", page[0].Username)

	page, total, err = repo.List(context.Background(), 0, 10, identity.SortUsername, false, "alp")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, page, 1)
	assert.Equal(t, "alpha", page[0].Username)

	require.NoError(t, repo.RecordLastLogin(context.Background(), "zeta", now.Add(time.Hour)))
	page, _, err = repo.List(context.Background(), 0, 1, identity.SortLastLogin, true, "")
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "zeta", page[0].Username)
}

func TestUserRepoProfileAndUniqueLoginName(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.Create(context.Background(), identity.User{
		ID:           "ada",
		Username:     "ada",
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        "ada@example.com",
		PasswordHash: "hash",
		Roles:        []string{identity.RoleAdmin},
		CreatedAt:    now,
		UpdatedAt:    now,
	}))

	got, err := repo.GetByUsername(context.Background(), "ADA")
	require.NoError(t, err)
	assert.Equal(t, "Ada", got.FirstName)
	assert.Equal(t, "Lovelace", got.LastName)
	assert.Equal(t, "ada@example.com", got.Email)

	err = repo.Create(context.Background(), identity.User{
		ID:           "other",
		Username:     "Ada",
		PasswordHash: "hash",
		Roles:        []string{identity.RoleReader},
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	assert.Error(t, err)
}

func TestUserRepoRecordLastLogin(t *testing.T) {
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "id.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	repo, err := New(st.DB())
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	u := identity.User{
		ID:                 "u1",
		Username:           "admin",
		PasswordHash:       "$argon2id$v=19$m=16,t=1,p=1$YWFhYWFhYWFhYWFhYWFhYQ$YmJiYmJiYmJiYmJiYmJiYg",
		Roles:              []string{identity.RoleAdmin},
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, repo.Create(context.Background(), u))

	got, err := repo.GetByID(context.Background(), "u1")
	require.NoError(t, err)
	assert.Nil(t, got.LastLogin)

	loginAt := now.Add(2 * time.Minute)
	require.NoError(t, repo.RecordLastLogin(context.Background(), "u1", loginAt))

	got.MustChangePassword = false
	got.UpdatedAt = now.Add(time.Minute)
	require.NoError(t, repo.Update(context.Background(), *got))

	byID, err := repo.GetByID(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, byID.LastLogin)
	assert.True(t, byID.LastLogin.Equal(loginAt))
	assert.False(t, byID.MustChangePassword)

	err = repo.RecordLastLogin(context.Background(), "missing", loginAt)
	assert.ErrorIs(t, err, identity.ErrUserNotFound)
}
