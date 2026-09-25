package idp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	domainidp "github.com/willie68/arcivio/internal/domain/idp"
)

func pkce() (verifier, challenge string) {
	verifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~ab"
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func TestHTTPAuthorizeLoginToken(t *testing.T) {
	prov := newMockIdpProvider(t)
	verifier, challenge := pkce()
	prov.EXPECT().Discovery().Return(map[string]any{
		"authorization_endpoint": "https://arcivio.test/auth/authorize",
	}).Once()
	prov.EXPECT().StartAuthorization(domainidp.AuthorizationRequest{
		ClientID:            domainidp.DefaultClientID,
		RedirectURI:         "https://arcivio.test/callback",
		ResponseType:        "code",
		Scope:               "openid",
		State:               "s1",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	}).Return("req-1", "/login", nil).Once()
	prov.EXPECT().CompleteLogin(mock.Anything, "req-1", "admin", "admin").
		Return(domainidp.LoginResult{Status: domainidp.LoginPasswordChange}, nil).Once()
	prov.EXPECT().CompletePasswordChange(mock.Anything, "req-1", "admin", "new-secret").
		Return(domainidp.LoginResult{
			Status:     domainidp.LoginOK,
			RedirectTo: "https://arcivio.test/callback?code=auth-code&state=s1",
		}, nil).Once()
	prov.EXPECT().ExchangeToken(mock.Anything, domainidp.TokenRequest{
		GrantType:    "authorization_code",
		Code:         "auth-code",
		RedirectURI:  "https://arcivio.test/callback",
		ClientID:     domainidp.DefaultClientID,
		CodeVerifier: verifier,
	}).Return(&domainidp.TokenResponse{AccessToken: "access-token", TokenType: "Bearer"}, nil).Once()
	prov.EXPECT().UserInfoFromAccessToken(mock.Anything, "access-token").
		Return(&domainidp.UserInfo{PreferredUsername: "admin"}, nil).Once()

	h := &Handler{idp: prov}
	r := chi.NewRouter()
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
	prov := newMockIdpProvider(t)
	prov.EXPECT().StartAuthorization(domainidp.AuthorizationRequest{
		ClientID:     domainidp.DefaultClientID,
		RedirectURI:  "https://arcivio.test/callback",
		ResponseType: "code",
		Scope:        "openid",
	}).Return("", "", domainidp.ErrInvalidRequest).Once()

	h := &Handler{idp: prov}
	router := chi.NewRouter()
	router.Mount(h.Routes())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/authorize?client_id="+domainidp.DefaultClientID+"&redirect_uri=https://arcivio.test/callback&response_type=code&scope=openid", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCookieSecureFollowsForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/authorize", nil)
	assert.False(t, cookieSecure(req))
	req.Header.Set("X-Forwarded-Proto", "http")
	assert.False(t, cookieSecure(req))
	req.Header.Set("X-Forwarded-Proto", "https")
	assert.True(t, cookieSecure(req))
}
