package identity

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// userStore is the generated UserStore mock with the in-memory behaviour the service tests rely on.
func userStore(t *testing.T) *mockUserStore {
	t.Helper()
	st := newMockUserStore(t)
	users := map[string]User{}
	var mu sync.Mutex

	st.EXPECT().Count(mock.Anything).RunAndReturn(func(context.Context) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return len(users), nil
	}).Maybe()
	st.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, id string) (*User, error) {
		mu.Lock()
		defer mu.Unlock()
		for _, u := range users {
			if u.ID == id {
				cp := u
				return &cp, nil
			}
		}
		return nil, ErrUserNotFound
	}).Maybe()
	st.EXPECT().GetByUsername(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, username string) (*User, error) {
		mu.Lock()
		defer mu.Unlock()
		u, ok := users[username]
		if !ok {
			return nil, ErrUserNotFound
		}
		cp := u
		return &cp, nil
	}).Maybe()
	st.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, user User) error {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := users[user.Username]; ok {
			return ErrAlreadyExists
		}
		users[user.Username] = user
		return nil
	}).Maybe()
	st.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, user User) error {
		mu.Lock()
		defer mu.Unlock()
		oldKey := ""
		for key, existing := range users {
			if existing.ID == user.ID {
				oldKey = key
				break
			}
		}
		if oldKey == "" {
			return ErrUserNotFound
		}
		if other, ok := users[user.Username]; ok && other.ID != user.ID {
			return ErrAlreadyExists
		}
		if oldKey != user.Username {
			delete(users, oldKey)
		}
		users[user.Username] = user
		return nil
	}).Maybe()
	st.EXPECT().Delete(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		for key, u := range users {
			if u.ID == id {
				delete(users, key)
				return nil
			}
		}
		return ErrUserNotFound
	}).Maybe()
	st.EXPECT().CountWithRole(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, role string) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, u := range users {
			if HasRole(&u, role) {
				n++
			}
		}
		return n, nil
	}).Maybe()
	st.EXPECT().List(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, offset, limit int, sortField string, desc bool, prefix string) ([]User, int, error) {
			mu.Lock()
			defer mu.Unlock()
			all := make([]User, 0, len(users))
			for _, u := range users {
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
		},
	).Maybe()
	st.EXPECT().RecordLastLogin(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, userID string, at time.Time) error {
		mu.Lock()
		defer mu.Unlock()
		for key, u := range users {
			if u.ID == userID {
				loggedIn := at
				u.LastLogin = &loggedIn
				users[key] = u
				return nil
			}
		}
		return ErrUserNotFound
	}).Maybe()
	return st
}

func TestBootstrapOnce(t *testing.T) {
	st := userStore(t)
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
	st := userStore(t)
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
	st := userStore(t)
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	_, err = svc.Authenticate(context.Background(), "admin", "nope")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = svc.Authenticate(context.Background(), "nobody", "admin")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePasswordClearsFlag(t *testing.T) {
	st := userStore(t)
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
	st := userStore(t)
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
	st := userStore(t)
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

func TestUpdateUserKeepsPasswordAndProfile(t *testing.T) {
	st := userStore(t)
	svc := New(st, testHasher())
	u, plain, err := svc.CreateUser(context.Background(), NewUser{
		Username: "clerk1", FirstName: "Alt", LastName: "Name", Email: "alt@example.com", Roles: []string{RoleClerk},
	})
	require.NoError(t, err)

	updated, err := svc.UpdateUser(context.Background(), u.ID, UserPatch{
		Username: "clerk2", FirstName: "Neu", LastName: "Berg", Email: "neu@example.com", Roles: []string{RoleReader, RoleClerk},
	})
	require.NoError(t, err)
	assert.Equal(t, "clerk2", updated.Username)
	assert.Equal(t, "Neu", updated.FirstName)
	assert.Equal(t, "Berg", updated.LastName)
	assert.Equal(t, "neu@example.com", updated.Email)
	assert.Equal(t, []string{RoleReader, RoleClerk}, updated.Roles)
	assert.Equal(t, u.PasswordHash, updated.PasswordHash)
	assert.True(t, updated.MustChangePassword)

	_, err = st.GetByUsername(context.Background(), "clerk1")
	assert.ErrorIs(t, err, ErrUserNotFound)
	got, err := svc.Authenticate(context.Background(), "clerk2", plain)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, _, err = svc.CreateUser(context.Background(), NewUser{Username: "admin", Roles: []string{RoleAdmin}})
	require.NoError(t, err)
	_, err = svc.UpdateUser(context.Background(), u.ID, UserPatch{Username: "admin", Roles: []string{RoleReader}})
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestUpdateUserRefusesLastAdminRole(t *testing.T) {
	st := userStore(t)
	svc := New(st, testHasher())
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	admin, err := st.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	_, err = svc.UpdateUser(context.Background(), admin.ID, UserPatch{Username: "admin", Roles: []string{RoleReader}})
	assert.ErrorIs(t, err, ErrLastAdmin)
}

func TestUpdateProfileLeavesLoginAndRoles(t *testing.T) {
	st := userStore(t)
	svc := New(st, testHasher())
	u, _, err := svc.CreateUser(context.Background(), NewUser{Username: "reader1", Roles: []string{RoleReader}})
	require.NoError(t, err)
	updated, err := svc.UpdateProfile(context.Background(), u.ID, ProfilePatch{
		FirstName: "Ida", LastName: "Horn", Email: "ida@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "reader1", updated.Username)
	assert.Equal(t, []string{RoleReader}, updated.Roles)
	assert.Equal(t, "Ida", updated.FirstName)
	assert.Equal(t, "ida@example.com", updated.Email)
	_, err = svc.UpdateProfile(context.Background(), u.ID, ProfilePatch{Email: "keine-mail"})
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestResetPasswordForcesChange(t *testing.T) {
	st := userStore(t)
	svc := New(st, testHasher())
	u, plain, err := svc.CreateUser(context.Background(), NewUser{Username: "clerk1", Roles: []string{RoleClerk}})
	require.NoError(t, err)
	_, err = svc.ChangePassword(context.Background(), u.ID, plain, "new-secret")
	require.NoError(t, err)

	reset, next, err := svc.ResetPassword(context.Background(), u.ID)
	require.NoError(t, err)
	assert.True(t, reset.MustChangePassword)
	assert.NotEqual(t, plain, next)
	assert.NotContains(t, reset.PasswordHash, next)
	_, err = svc.Authenticate(context.Background(), "clerk1", "new-secret")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	got, err := svc.Authenticate(context.Background(), "clerk1", next)
	require.NoError(t, err)
	assert.True(t, got.MustChangePassword)
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
