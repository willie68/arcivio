package identity

// Sort fields accepted by UserStore.List.
const (
	SortUsername   = "username"
	SortFirstName  = "firstName"
	SortLastName   = "lastName"
	SortEmail      = "email"
	SortRoles      = "roles"
	SortLastLogin  = "lastLogin"
	SortMustChange = "mustChangePassword"
)

// CanonicalUserSort maps a client sort field to a known column.
// ok is false when field is empty or unknown; callers then use username ascending.
func CanonicalUserSort(field string) (string, bool) {
	switch field {
	case SortUsername, SortFirstName, SortLastName, SortEmail, SortRoles, SortLastLogin, SortMustChange:
		return field, true
	default:
		return SortUsername, false
	}
}
