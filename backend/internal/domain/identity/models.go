package identity

import (
	"errors"
	"time"
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

// UserPatch is an admin edit of a local account. The password is not part of it.
type UserPatch struct {
	Username  string
	FirstName string
	LastName  string
	Email     string
	Roles     []string
}

// NewUser is the input for creating a local account.
type NewUser struct {
	Username  string
	FirstName string
	LastName  string
	Email     string
	Roles     []string
}

// ProfilePatch is what a user may change on their own account.
type ProfilePatch struct {
	FirstName string
	LastName  string
	Email     string
}

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
