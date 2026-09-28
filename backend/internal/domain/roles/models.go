package roles

import "errors"

var (
	ErrInvalidRole = errors.New("invalid role")
)

// Built-in RBAC roles from PLAN.md.
const (
	RoleAdmin     = "admin"
	RoleArchivist = "archivist"
	RoleClerk     = "clerk"
	RoleReader    = "reader"
)

// Text is a short UI string in German and English.
type Text struct {
	De string
	En string
}

// RoleDefinition is one built-in RBAC role as shown in settings.
type RoleDefinition struct {
	Name        string
	Labels      Text
	Description Text
}
