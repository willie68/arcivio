package identity

import (
	"strings"
	"unicode/utf8"
)

// MinPrefixLen is the shortest filter that narrows the user list.
const MinPrefixLen = 3

// NormalizePrefix returns a trimmed prefix, or empty when it is shorter than MinPrefixLen.
func NormalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if utf8.RuneCountInString(prefix) < MinPrefixLen {
		return ""
	}
	return prefix
}

// MatchPrefix reports whether user matches a prefix on login name, first name, last name or email.
// A prefix shorter than MinPrefixLen matches every user.
func MatchPrefix(user User, prefix string) bool {
	prefix = strings.ToLower(NormalizePrefix(prefix))
	if prefix == "" {
		return true
	}
	for _, value := range []string{user.Username, user.FirstName, user.LastName, user.Email} {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), prefix) {
			return true
		}
	}
	return false
}
