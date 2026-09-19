package idp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPKCERoundTrip(t *testing.T) {
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	assert.Len(t, verifier, 66)
	ch := challengeS256(verifier)
	assert.NoError(t, verifyPKCE(verifier, ch, pkceS256))
	assert.Error(t, verifyPKCE(verifier+"x", ch, pkceS256))
	assert.Error(t, verifyPKCE(verifier, ch, "plain"))
	assert.Error(t, verifyPKCE("tooshort", ch, pkceS256))
}

func TestRandomURLToken(t *testing.T) {
	a, err := randomURLToken(32)
	require.NoError(t, err)
	b, err := randomURLToken(32)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.GreaterOrEqual(t, len(a), 40)
}
