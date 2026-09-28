package roles

type service struct {
	roles map[string]RoleDefinition
}

func NewService() *service {
	roles := make(map[string]RoleDefinition)
	for _, role := range builtinRoles {
		roles[role.Name] = role
	}
	return &service{roles: roles}
}

func (s *service) GetRoles() []RoleDefinition {
	out := make([]RoleDefinition, len(builtinRoles))
	copy(out, builtinRoles)
	return out
}

func (s *service) GetRole(name string) *RoleDefinition {
	role, ok := s.roles[name]
	if !ok {
		return nil
	}
	return &role
}

func (s *service) HasRole(checkRole string, desiredRole string) bool {
	role, ok := s.roles[checkRole]
	if !ok {
		return false
	}
	return role.Name == desiredRole
}

// ValidRole reports whether role is one of the built-in RBAC roles.
func (s *service) ValidRole(role string) bool {
	_, ok := s.roles[role]
	return ok
}

// ValidateRoles returns ErrInvalidRole if any role is unknown.
func (s *service) ValidateRoles(roles []string) error {
	if len(roles) == 0 {
		return ErrInvalidRole
	}
	for _, r := range roles {
		if !s.ValidRole(r) {
			return ErrInvalidRole
		}
	}
	return nil
}
