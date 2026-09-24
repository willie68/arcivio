package identity

import (
	"sort"
	"strings"
	"time"
)

// sortUsers orders users like the SQLite list: requested column, then username.
// Empty last-login values sort before any timestamp.
func sortUsers(users []User, field string, desc bool) {
	if canonical, ok := CanonicalUserSort(field); ok {
		field = canonical
	} else {
		field = SortUsername
		desc = false
	}
	sort.SliceStable(users, func(i, j int) bool {
		return userBefore(users[i], users[j], field, desc)
	})
}

func userBefore(a, b User, field string, desc bool) bool {
	cmp := compareUsers(a, b, field)
	if cmp == 0 {
		return strings.ToLower(a.Username) < strings.ToLower(b.Username)
	}
	if desc {
		return cmp > 0
	}
	return cmp < 0
}

func compareUsers(a, b User, field string) int {
	switch field {
	case SortFirstName:
		return strings.Compare(strings.ToLower(a.FirstName), strings.ToLower(b.FirstName))
	case SortLastName:
		return strings.Compare(strings.ToLower(a.LastName), strings.ToLower(b.LastName))
	case SortEmail:
		return strings.Compare(strings.ToLower(a.Email), strings.ToLower(b.Email))
	case SortRoles:
		return strings.Compare(strings.ToLower(strings.Join(a.Roles, ",")), strings.ToLower(strings.Join(b.Roles, ",")))
	case SortLastLogin:
		return compareTime(a.LastLogin, b.LastLogin)
	case SortMustChange:
		return compareBool(a.MustChangePassword, b.MustChangePassword)
	default:
		return strings.Compare(strings.ToLower(a.Username), strings.ToLower(b.Username))
	}
}

func compareTime(a, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	case a.Before(*b):
		return -1
	case a.After(*b):
		return 1
	default:
		return 0
	}
}

func compareBool(a, b bool) int {
	if a == b {
		return 0
	}
	if !a {
		return -1
	}
	return 1
}
