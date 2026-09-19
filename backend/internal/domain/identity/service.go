package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/xid"
)

const (
	bootstrapUsername = "admin"
	bootstrapPassword = "admin"
	minPasswordLength = 8
)

// Service is the identity use-case root (local accounts, bootstrap, password rules).
type Service struct {
	users  UserStore
	hasher PasswordHasher
	now    func() time.Time
}

// New creates the identity service.
func New(users UserStore, hasher PasswordHasher) *Service {
	if hasher == nil {
		hasher = NewArgon2Hasher()
	}
	return &Service{
		users:  users,
		hasher: hasher,
		now:    time.Now,
	}
}

// Bootstrap creates the first admin/admin account when the user table is empty.
// It never creates a second automatic user. created is true only when the
// bootstrap account was inserted in this call.
func (s *Service) Bootstrap(ctx context.Context) (created bool, err error) {
	n, err := s.users.Count(ctx)
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	hash, err := s.hasher.Hash(bootstrapPassword)
	if err != nil {
		return false, err
	}
	now := s.now().UTC()
	u := User{
		ID:                 xid.New().String(),
		Username:           bootstrapUsername,
		PasswordHash:       hash,
		Roles:              []string{RoleAdmin},
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return false, fmt.Errorf("bootstrap admin: %w", err)
	}
	return true, nil
}

// Authenticate verifies username and password. It does not issue tokens.
func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	username = normalizeUsername(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	u, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := s.hasher.Verify(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// GetByID loads a user by id.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	if id == "" {
		return nil, ErrUserNotFound
	}
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ChangePassword verifies the old password and stores a new Argon2id hash.
// On success MustChangePassword is cleared.
func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) (*User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.hasher.Verify(u.PasswordHash, oldPassword); err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := ValidateNewPassword(newPassword, oldPassword); err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash
	u.MustChangePassword = false
	u.UpdatedAt = s.now().UTC()
	if err := s.users.Update(ctx, *u); err != nil {
		return nil, err
	}
	return u, nil
}

// CreateUser creates a local user with a random one-time password.
// The plaintext password is returned once and not persisted.
func (s *Service) CreateUser(ctx context.Context, username string, roles []string) (*User, string, error) {
	username = normalizeUsername(username)
	if username == "" {
		return nil, "", fmt.Errorf("username required")
	}
	if err := ValidateRoles(roles); err != nil {
		return nil, "", err
	}
	if existing, err := s.users.GetByUsername(ctx, username); err == nil && existing != nil {
		return nil, "", ErrAlreadyExists
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, "", err
	}
	plain, err := randomSecret(18)
	if err != nil {
		return nil, "", err
	}
	hash, err := s.hasher.Hash(plain)
	if err != nil {
		return nil, "", err
	}
	now := s.now().UTC()
	u := User{
		ID:                 xid.New().String(),
		Username:           username,
		PasswordHash:       hash,
		Roles:              append([]string(nil), roles...),
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, "", err
	}
	return &u, plain, nil
}

// ValidateNewPassword enforces length and that the password actually changed.
func ValidateNewPassword(newPassword, oldPassword string) error {
	if utf8.RuneCountInString(newPassword) < minPasswordLength {
		return ErrWeakPassword
	}
	if newPassword == oldPassword {
		return ErrSamePassword
	}
	return nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
