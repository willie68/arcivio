package roles

// Service is the built-in role catalog.
type Service struct {
	roles map[string]RoleDefinition
}

// NewService returns the built-in role catalog.
func NewService() *Service {
	roles := make(map[string]RoleDefinition)
	for _, role := range builtinRoles {
		roles[role.Name] = role
	}
	return &Service{roles: roles}
}

func (s *service) GetRoles() []RoleDefinition {
	out := make([]RoleDefinition, len(s.roles))
	i := 0
	for _, role := range s.roles {
		out[i] = role
		i++
	}
}
	return out
}

// GetRole returns the definition for name, or nil when the name is unknown.
func (s *Service) GetRole(name string) *RoleDefinition {
	role, ok := s.roles[name]
	if !ok {
		return nil
	}
	return &role
}

// HasRole reports whether checkRole is the built-in role desiredRole.
func (s *Service) HasRole(checkRole string, desiredRole string) bool {
	role, ok := s.roles[checkRole]
	if !ok {
		return false
	}
	return role.Name == desiredRole
}

// ValidRole reports whether role is one of the built-in RBAC roles.
func (s *Service) ValidRole(role string) bool {
	_, ok := s.roles[role]
	return ok
}

// ValidateRoles returns ErrInvalidRole if any role is unknown.
func (s *Service) ValidateRoles(roles []string) error {
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
