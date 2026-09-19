package identity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHasher() *Argon2Hasher {
	return NewArgon2HasherWithParams(1, 16*1024, 1, 32, 16)
}

func TestArgon2HashIsNotPlaintext(t *testing.T) {
	h := testHasher()
	encoded, err := h.Hash("admin")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(encoded, "$argon2id$"))
	assert.NotContains(t, encoded, "admin")
	assert.NoError(t, h.Verify(encoded, "admin"))
	assert.ErrorIs(t, h.Verify(encoded, "wrong"), ErrInvalidCredentials)
	assert.ErrorIs(t, h.Verify("not-a-hash", "admin"), ErrInvalidCredentials)
}

func TestArgon2DifferentSalts(t *testing.T) {
	h := testHasher()
	a, err := h.Hash("secret-pass")
	require.NoError(t, err)
	b, err := h.Hash("secret-pass")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestValidateNewPassword(t *testing.T) {
	assert.ErrorIs(t, ValidateNewPassword("short", "old"), ErrWeakPassword)
	assert.ErrorIs(t, ValidateNewPassword("samepass", "samepass"), ErrSamePassword)
	assert.NoError(t, ValidateNewPassword("new-password", "old-password"))
}

func TestValidRole(t *testing.T) {
	assert.True(t, ValidRole(RoleAdmin))
	assert.True(t, ValidRole(RoleReader))
	assert.False(t, ValidRole("superuser"))
	assert.ErrorIs(t, ValidateRoles(nil), ErrInvalidRole)
	assert.NoError(t, ValidateRoles([]string{RoleClerk, RoleArchivist}))
}
