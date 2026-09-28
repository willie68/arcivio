package roles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRolesReturnsCatalogCopy(t *testing.T) {
	s := NewService()
	listed := s.GetRoles()
	require.Len(t, listed, 4)
	assert.Equal(t, []string{RoleAdmin, RoleArchivist, RoleClerk, RoleReader}, names(listed))
	for _, role := range listed {
		assert.NotEmpty(t, role.Labels.De)
		assert.NotEmpty(t, role.Labels.En)
		assert.NotEmpty(t, role.Description.De)
		assert.NotEmpty(t, role.Description.En)
	}

	listed[0].Labels.De = "changed"
	again := s.GetRoles()
	assert.Equal(t, "Administrator", again[0].Labels.De)
}

func TestGetRole(t *testing.T) {
	s := NewService()
	admin := s.GetRole(RoleAdmin)
	require.NotNil(t, admin)
	assert.Equal(t, RoleAdmin, admin.Name)
	assert.Equal(t, "Administrator", admin.Labels.De)
	assert.Equal(t, "Administrator", admin.Labels.En)

	admin.Labels.En = "changed"
	again := s.GetRole(RoleAdmin)
	require.NotNil(t, again)
	assert.Equal(t, "Administrator", again.Labels.En)
	assert.Nil(t, s.GetRole("superuser"))
	assert.Nil(t, s.GetRole(""))
}

func TestHasRole(t *testing.T) {
	s := NewService()
	assert.True(t, s.HasRole(RoleAdmin, RoleAdmin))
	assert.False(t, s.HasRole(RoleClerk, RoleAdmin))
	assert.False(t, s.HasRole("superuser", RoleAdmin))
	assert.False(t, s.HasRole(RoleAdmin, "superuser"))
}

func TestValidRole(t *testing.T) {
	s := NewService()
	for _, name := range []string{RoleAdmin, RoleArchivist, RoleClerk, RoleReader} {
		assert.True(t, s.ValidRole(name))
	}
	assert.False(t, s.ValidRole("superuser"))
	assert.False(t, s.ValidRole(""))
}

func TestValidateRoles(t *testing.T) {
	s := NewService()
	assert.NoError(t, s.ValidateRoles([]string{RoleClerk, RoleArchivist}))
	assert.ErrorIs(t, s.ValidateRoles(nil), ErrInvalidRole)
	assert.ErrorIs(t, s.ValidateRoles([]string{}), ErrInvalidRole)
	assert.ErrorIs(t, s.ValidateRoles([]string{RoleAdmin, "superuser"}), ErrInvalidRole)
}

func names(listed []RoleDefinition) []string {
	out := make([]string, len(listed))
	for i, role := range listed {
		out[i] = role.Name
	}
	return out
}
