package identity

import (
	"context"
	"errors"
	"time"
)

// Built-in RBAC roles from PLAN.md.
const (
	RoleAdmin     = "admin"
	RoleArchivist = "archivist"
	RoleClerk     = "clerk"
	RoleReader    = "reader"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrAlreadyExists      = errors.New("user already exists")
	ErrWeakPassword       = errors.New("password does not meet requirements")
	ErrSamePassword       = errors.New("new password must differ from the old password")
	ErrInvalidRole        = errors.New("invalid role")
)

// User is a local identity (internal IdP account).
type User struct {
	ID                 string
	Username           string
	PasswordHash       string
	Roles              []string
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// UserStore is the outbound port for local users.
type UserStore interface {
	Count(ctx context.Context) (int, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Create(ctx context.Context, user User) error
	Update(ctx context.Context, user User) error
}

// PasswordHasher hashes and verifies passwords (Argon2id).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(encodedHash, password string) error
}

// ValidRole reports whether role is one of the built-in RBAC roles.
func ValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleArchivist, RoleClerk, RoleReader:
		return true
	default:
		return false
	}
}

// ValidateRoles returns ErrInvalidRole if any role is unknown.
func ValidateRoles(roles []string) error {
	if len(roles) == 0 {
		return ErrInvalidRole
	}
	for _, r := range roles {
		if !ValidRole(r) {
			return ErrInvalidRole
		}
	}
	return nil
}
