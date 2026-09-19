package idp

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/identity"
)

func testHasher() *identity.Argon2Hasher {
	return identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16)
}

type memStore struct {
	mu    sync.Mutex
	users map[string]identity.User
}

func newMemStore() *memStore {
	return &memStore{users: make(map[string]identity.User)}
}

func (m *memStore) Count(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.users), nil
}

func (m *memStore) GetByID(_ context.Context, id string) (*identity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, identity.ErrUserNotFound
}

func (m *memStore) GetByUsername(_ context.Context, username string) (*identity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[username]
	if !ok {
		return nil, identity.ErrUserNotFound
	}
	cp := u
	return &cp, nil
}

func (m *memStore) Create(_ context.Context, user identity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Username] = user
	return nil
}

func (m *memStore) Update(_ context.Context, user identity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Username] = user
	return nil
}

func newTestIDP(t *testing.T) (*Provider, *identity.Service) {
	t.Helper()
	st := newMemStore()
	ident := identity.New(st, testHasher())
	_, err := ident.Bootstrap(context.Background())
	require.NoError(t, err)
	// bootstrap user must change password; for token tests we change it first
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
