package idp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willie68/arcivio/internal/domain/identity"
	domainidp "github.com/willie68/arcivio/internal/domain/idp"
)

func pkce() (verifier, challenge string) {
	verifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~ab"
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func TestHTTPAuthorizeLoginToken(t *testing.T) {
	st := newMemStore()
	ident := identity.New(st, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16))
	_, err := ident.Bootstrap(context.Background())
	require.NoError(t, err)

	prov, err := domainidp.New(domainidp.Config{
		Issuer:       "https://arcivio.test/auth",
		ClientID:     domainidp.DefaultClientID,
		Audience:     domainidp.DefaultClientID,
		RedirectURIs: []string{"https://arcivio.test/callback"},
		LoginPath:    "/login",
	}, ident)
	require.NoError(t, err)
	t.Cleanup(func() { _ = prov.Shutdown() })

	r := chi.NewRouter()
	h := New(prov)
	r.Mount(h.Routes())
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("login"))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/.well-known/openid-configuration", nil)
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"authorization_endpoint"`)

	verifier, challenge := pkce()
	q := url.Values{
		"client_id":             {domainidp.DefaultClientID},
		"redirect_uri":          {"https://arcivio.test/callback"},
		"response_type":         {"code"},
		"scope":                 {"openid"},
		"state":                 {"s1"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	resp, err := client.Get(srv.URL + "/auth/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	loc := resp.Header.Get("Location")
	assert.Equal(t, "/login", loc)

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	resp, err = client.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	var login domainidp.LoginResult
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&login))
	assert.Equal(t, domainidp.LoginPasswordChange, login.Status)

	body, _ = json.Marshal(map[string]string{"oldPassword": "admin", "newPassword": "new-secret"})
	resp, err = client.Post(srv.URL+"/auth/change-password", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&login))
	require.Equal(t, domainidp.LoginOK, login.Status)

	u, err := url.Parse(login.RedirectTo)
	require.NoError(t, err)
	code := u.Query().Get("code")
	require.NotEmpty(t, code)

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"https://arcivio.test/callback"},
		"client_id":     {domainidp.DefaultClientID},
		"code_verifier": {verifier},
	}
	resp, err = client.Post(srv.URL+"/auth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var tok domainidp.TokenResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&tok))
	assert.NotEmpty(t, tok.AccessToken)

	req = httptest.NewRequest(http.MethodGet, "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"preferred_username":"admin"`)
}

func TestAuthorizeRejectsMissingPKCE(t *testing.T) {
	st := newMemStore()
	ident := identity.New(st, identity.NewArgon2HasherWithParams(1, 8*1024, 1, 32, 16))
	_, err := ident.Bootstrap(context.Background())
	require.NoError(t, err)
	prov, err := domainidp.New(domainidp.Config{
		Issuer:       "https://arcivio.test/auth",
		ClientID:     domainidp.DefaultClientID,
		Audience:     domainidp.DefaultClientID,
		RedirectURIs: []string{"https://arcivio.test/callback"},
	}, ident)
	require.NoError(t, err)
	t.Cleanup(func() { _ = prov.Shutdown() })

	h := New(prov)
	router := chi.NewRouter()
	router.Mount(h.Routes())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/authorize?client_id="+domainidp.DefaultClientID+"&redirect_uri=https://arcivio.test/callback&response_type=code&scope=openid", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

type memStore struct {
	users map[string]identity.User
}

func newMemStore() *memStore {
	return &memStore{users: make(map[string]identity.User)}
}

func (m *memStore) Count(_ context.Context) (int, error) { return len(m.users), nil }

func (m *memStore) GetByID(_ context.Context, id string) (*identity.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, identity.ErrUserNotFound
}

func (m *memStore) GetByUsername(_ context.Context, username string) (*identity.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, identity.ErrUserNotFound
	}
	cp := u
	return &cp, nil
}

func (m *memStore) Create(_ context.Context, user identity.User) error {
	m.users[user.Username] = user
	return nil
}

func (m *memStore) Update(_ context.Context, user identity.User) error {
	m.users[user.Username] = user
	return nil
}

func (m *memStore) RecordLastLogin(_ context.Context, userID string, at time.Time) error {
	for key, u := range m.users {
		if u.ID == userID {
			t := at
			u.LastLogin = &t
			m.users[key] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func TestCookieSecureFollowsForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/authorize", nil)
	assert.False(t, cookieSecure(req))
	req.Header.Set("X-Forwarded-Proto", "http")
	assert.False(t, cookieSecure(req))
	req.Header.Set("X-Forwarded-Proto", "https")
	assert.True(t, cookieSecure(req))
}
