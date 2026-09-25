package idp

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/identity"
)

func newTestIDP(t *testing.T) (*Provider, *mockIdentityService) {
	t.Helper()
	ident := newMockIdentityService(t)
	cfg := Config{
		Issuer:         "https://arcivio.test/auth",
		ClientID:       DefaultClientID,
		Audience:       DefaultClientID,
		RedirectURIs:   []string{"https://arcivio.test/callback"},
		LoginPath:      "/login",
		AccessTokenTTL: time.Hour,
		IDTokenTTL:     time.Hour,
		AuthCodeTTL:    5 * time.Minute,
		AuthRequestTTL: 5 * time.Minute,
	}
	p, err := New(cfg, ident)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown() })
	return p, ident
}

func mustChangeAdmin() *identity.User {
	return &identity.User{
		ID:                 "admin-id",
		Username:           "admin",
		Roles:              []string{identity.RoleAdmin},
		MustChangePassword: true,
	}
}

func readyAdmin() *identity.User {
	return &identity.User{
		ID:       "admin-id",
		Username: "admin",
		Roles:    []string{identity.RoleAdmin},
	}
}

func pkcePair() (verifier, challenge string) {
	verifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~ab"
	return verifier, challengeS256(verifier)
}

func TestAuthorizeRequiresPKCE(t *testing.T) {
	p, _ := newTestIDP(t)
	_, _, err := p.StartAuthorization(AuthorizationRequest{
		ClientID:     DefaultClientID,
		RedirectURI:  "https://arcivio.test/callback",
		ResponseType: "code",
		Scope:        "openid",
	})
	assert.ErrorIs(t, err, ErrInvalidRequest)

	id, login, err := p.StartAuthorization(AuthorizationRequest{
		ClientID:            DefaultClientID,
		RedirectURI:         "https://arcivio.test/callback",
		ResponseType:        "code",
		Scope:               "openid profile",
		State:               "st",
		CodeChallenge:       "abc",
		CodeChallengeMethod: pkceS256,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "/login", login)
}

func TestCodeFlowPKCEAndMustChangePassword(t *testing.T) {
	p, ident := newTestIDP(t)
	ctx := context.Background()
	ident.EXPECT().Authenticate(mock.Anything, "admin", "admin").Return(mustChangeAdmin(), nil).Once()
	ident.EXPECT().ChangePassword(mock.Anything, "admin-id", "admin", "new-secret").Return(readyAdmin(), nil).Once()
	ident.EXPECT().GetByID(mock.Anything, "admin-id").Return(readyAdmin(), nil).Twice()
	ident.EXPECT().Authenticate(mock.Anything, "admin", "new-secret").Return(readyAdmin(), nil).Once()
	verifier, challenge := pkcePair()

	reqID, _, err := p.StartAuthorization(AuthorizationRequest{
		ClientID:            DefaultClientID,
		RedirectURI:         "https://arcivio.test/callback",
		ResponseType:        "code",
		Scope:               "openid",
		State:               "xyz",
		Nonce:               "n1",
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkceS256,
	})
	require.NoError(t, err)

	res, err := p.CompleteLogin(ctx, reqID, "admin", "admin")
	require.NoError(t, err)
	assert.Equal(t, LoginPasswordChange, res.Status)
	assert.Empty(t, res.RedirectTo)

	res, err = p.CompletePasswordChange(ctx, reqID, "admin", "new-secret")
	require.NoError(t, err)
	assert.Equal(t, LoginOK, res.Status)
	assert.Contains(t, res.RedirectTo, "code=")
	assert.Contains(t, res.RedirectTo, "state=xyz")

	code := extractQuery(t, res.RedirectTo, "code")

	_, err = p.ExchangeToken(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://arcivio.test/callback",
		ClientID:     DefaultClientID,
		CodeVerifier: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", // valid length, wrong value
	})
	assert.ErrorIs(t, err, ErrInvalidGrant)

	tok, err := p.ExchangeToken(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://arcivio.test/callback",
		ClientID:     DefaultClientID,
		CodeVerifier: verifier,
	})
	require.NoError(t, err)
	assert.Equal(t, "Bearer", tok.TokenType)
	assert.NotEmpty(t, tok.AccessToken)
	assert.NotEmpty(t, tok.IDToken)

	_, err = p.ExchangeToken(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://arcivio.test/callback",
		ClientID:     DefaultClientID,
		CodeVerifier: verifier,
	})
	assert.ErrorIs(t, err, ErrInvalidGrant)

	claims, err := p.VerifyAccessToken(tok.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "https://arcivio.test/auth", claims["iss"])
	info, err := p.UserInfoFromAccessToken(ctx, tok.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "admin", info.PreferredUsername)
	assert.False(t, info.MustChangePassword)

	u, err := ident.Authenticate(ctx, "admin", "new-secret")
	require.NoError(t, err)
	assert.False(t, u.MustChangePassword)
}

func TestJWTSignatureRequired(t *testing.T) {
	p, ident := newTestIDP(t)
	ctx := context.Background()
	ident.EXPECT().Authenticate(mock.Anything, "admin", "admin").Return(mustChangeAdmin(), nil).Once()
	ident.EXPECT().ChangePassword(mock.Anything, "admin-id", "admin", "new-secret").Return(readyAdmin(), nil).Once()
	ident.EXPECT().Authenticate(mock.Anything, "admin", "new-secret").Return(readyAdmin(), nil).Once()
	ident.EXPECT().GetByID(mock.Anything, "admin-id").Return(readyAdmin(), nil).Once()
	admin, err := ident.Authenticate(ctx, "admin", "admin")
	require.NoError(t, err)
	_, err = ident.ChangePassword(ctx, admin.ID, "admin", "new-secret")
	require.NoError(t, err)

	verifier, challenge := pkcePair()
	reqID, _, err := p.StartAuthorization(AuthorizationRequest{
		ClientID:            DefaultClientID,
		RedirectURI:         "https://arcivio.test/callback",
		ResponseType:        "code",
		Scope:               "openid",
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkceS256,
	})
	require.NoError(t, err)
	res, err := p.CompleteLogin(ctx, reqID, "admin", "new-secret")
	require.NoError(t, err)
	code := extractQuery(t, res.RedirectTo, "code")
	tok, err := p.ExchangeToken(ctx, TokenRequest{
		GrantType: "authorization_code", Code: code, RedirectURI: "https://arcivio.test/callback",
		ClientID: DefaultClientID, CodeVerifier: verifier,
	})
	require.NoError(t, err)

	tampered := tok.AccessToken[:len(tok.AccessToken)-4] + "xxxx"
	_, err = p.VerifyAccessToken(tampered)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func extractQuery(t *testing.T, raw, key string) string {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	v := u.Query().Get(key)
	require.NotEmpty(t, v)
	return v
}
