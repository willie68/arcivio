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
	ErrInvalidEmail       = errors.New("invalid email")
	ErrDeleteSelf         = errors.New("cannot delete the signed-in user")
	ErrLastAdmin          = errors.New("cannot delete the last admin")
)

// User is a local identity (internal IdP account).
// Username is the unique login name.
type User struct {
	ID                 string
	Username           string
	FirstName          string
	LastName           string
	Email              string
	PasswordHash       string
	Roles              []string
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastLogin          *time.Time
}

// UserStore is the outbound port for local users.
type UserStore interface {
	Count(ctx context.Context) (int, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Create(ctx context.Context, user User) error
	Update(ctx context.Context, user User) error
	RecordLastLogin(ctx context.Context, userID string, at time.Time) error
	Delete(ctx context.Context, id string) error
	CountWithRole(ctx context.Context, role string) (int, error)
	// List returns one page of users and the total count before paging.
	// sort is one of the Sort* fields; unknown values sort by username ascending.
	// prefix filters login name, first name, last name and email when it has at least three characters.
	List(ctx context.Context, offset, limit int, sort string, desc bool, prefix string) ([]User, int, error)
}

// HasRole reports whether user has role.
func HasRole(user *User, role string) bool {
	if user == nil {
		return false
	}
	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
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
