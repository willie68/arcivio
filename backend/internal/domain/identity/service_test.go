package identity

import (
	"context"
	"sync"
	"testing"
	"time"

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

func (m *memStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, u := range m.users {
		if u.ID == id {
			delete(m.users, key)
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *memStore) CountWithRole(_ context.Context, role string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, u := range m.users {
		if HasRole(&u, role) {
			n++
		}
	}
	return n, nil
}

func (m *memStore) List(_ context.Context, offset, limit int, sortField string, desc bool, prefix string) ([]User, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := make([]User, 0, len(m.users))
	for _, u := range m.users {
		if MatchPrefix(u, prefix) {
			all = append(all, u)
		}
	}
	sortUsers(all, sortField, desc)
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return append([]User(nil), all[offset:end]...), total, nil
}

func (m *memStore) RecordLastLogin(_ context.Context, userID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, u := range m.users {
		if u.ID == userID {
			t := at
			u.LastLogin = &t
			m.users[key] = u
			return nil
		}
	}
	return ErrUserNotFound
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

func TestAuthenticateRecordsLastLogin(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	fixed := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)

	before, err := st.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	assert.Nil(t, before.LastLogin)

	u, err := svc.Authenticate(context.Background(), "admin", "admin")
	require.NoError(t, err)
	require.NotNil(t, u.LastLogin)
	assert.True(t, u.LastLogin.Equal(fixed))

	stored, err := st.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	require.NotNil(t, stored.LastLogin)
	assert.True(t, stored.LastLogin.Equal(fixed))

	later := fixed.Add(time.Hour)
	svc.now = func() time.Time { return later }
	_, err = svc.Authenticate(context.Background(), "admin", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	stored, err = st.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	require.NotNil(t, stored.LastLogin)
	assert.True(t, stored.LastLogin.Equal(fixed))

	_, err = svc.Authenticate(context.Background(), "admin", "admin")
	require.NoError(t, err)
	stored, err = st.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	require.NotNil(t, stored.LastLogin)
	assert.True(t, stored.LastLogin.Equal(later))
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

func TestListUsersPagesByUsername(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	_, _, err := svc.CreateUser(context.Background(), NewUser{Username: "zeta", Roles: []string{RoleReader}})
	require.NoError(t, err)
	_, _, err = svc.CreateUser(context.Background(), NewUser{Username: "alpha", FirstName: "Ann", Roles: []string{RoleClerk}})
	require.NoError(t, err)
	_, _, err = svc.CreateUser(context.Background(), NewUser{Username: "mid", Roles: []string{RoleArchivist}})
	require.NoError(t, err)

	page, total, err := svc.ListUsers(context.Background(), 0, 2, SortUsername, false, "")
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page, 2)
	assert.Equal(t, "alpha", page[0].Username)
	assert.Equal(t, "mid", page[1].Username)

	page, total, err = svc.ListUsers(context.Background(), 0, 1, SortUsername, true, "")
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "zeta", page[0].Username)

	page, total, err = svc.ListUsers(context.Background(), 0, 1, "passwordHash", true, "")
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "alpha", page[0].Username)

	page, total, err = svc.ListUsers(context.Background(), 2, 2, SortUsername, false, "z")
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page, 1)
	assert.Equal(t, "zeta", page[0].Username)

	page, total, err = svc.ListUsers(context.Background(), 0, 10, SortUsername, false, "alp")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, page, 1)
	assert.Equal(t, "alpha", page[0].Username)

	page, total, err = svc.ListUsers(context.Background(), 0, 10, SortFirstName, false, "ann")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "alpha", page[0].Username)
}

func TestCreateUserRandomPasswordNotStored(t *testing.T) {
	st := newMemStore()
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)

	u, plain, err := svc.CreateUser(context.Background(), NewUser{Username: "clerk1", Roles: []string{RoleClerk}})
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

	_, _, err = svc.CreateUser(context.Background(), NewUser{Username: "clerk1", Roles: []string{RoleClerk}})
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
