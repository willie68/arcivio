package identity

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	mu    sync.Mutex
	users map[string]User
}

func newMemStore() *memStore {
	return &memStore{users: make(map[string]User)}
}

func (m *memStore) Count(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.users), nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *memStore) GetByUsername(_ context.Context, username string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := u
	return &cp, nil
}

func (m *memStore) Create(_ context.Context, user User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.Username]; ok {
		return ErrAlreadyExists
	}
	m.users[user.Username] = user
	return nil
}

func (m *memStore) Update(_ context.Context, user User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.Username]; !ok {
		return ErrUserNotFound
	}
	m.users[user.Username] = user
	return nil
}

func TestBootstrapOnce(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	created, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	assert.True(t, created)

	u, err := svc.Authenticate(context.Background(), "Admin", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Username)
	assert.True(t, u.MustChangePassword)
	assert.Equal(t, []string{RoleAdmin}, u.Roles)
	assert.True(t, stringsHasPrefix(u.PasswordHash, "$argon2id$"))
	assert.NotEqual(t, "admin", u.PasswordHash)

	created, err = svc.Bootstrap(context.Background())
	require.NoError(t, err)
	assert.False(t, created)
	n, err := st.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestAuthenticateRejectsWrongPassword(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	_, err = svc.Authenticate(context.Background(), "admin", "nope")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = svc.Authenticate(context.Background(), "nobody", "admin")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePasswordClearsFlag(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	u, err := svc.Authenticate(context.Background(), "admin", "admin")
	require.NoError(t, err)

	updated, err := svc.ChangePassword(context.Background(), u.ID, "admin", "new-secret")
	require.NoError(t, err)
	assert.False(t, updated.MustChangePassword)
	assert.NotEqual(t, u.PasswordHash, updated.PasswordHash)

	_, err = svc.Authenticate(context.Background(), "admin", "admin")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	got, err := svc.Authenticate(context.Background(), "admin", "new-secret")
	require.NoError(t, err)
	assert.False(t, got.MustChangePassword)
}

func TestCreateUserRandomPasswordNotStored(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)

	u, plain, err := svc.CreateUser(context.Background(), "clerk1", []string{RoleClerk})
	require.NoError(t, err)
	assert.NotEmpty(t, plain)
	assert.True(t, u.MustChangePassword)
	stored, err := st.GetByUsername(context.Background(), "clerk1")
	require.NoError(t, err)
	assert.NotEqual(t, plain, stored.PasswordHash)
	assert.NotContains(t, stored.PasswordHash, plain)

	got, err := svc.Authenticate(context.Background(), "clerk1", plain)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, _, err = svc.CreateUser(context.Background(), "clerk1", []string{RoleClerk})
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
