package idp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/utils/ttlcache"
)

type identityService interface {
	ListUsers(ctx context.Context, offset, limit int, sort string, desc bool, prefix string) ([]identity.User, int, error)
	GetByID(ctx context.Context, id string) (*identity.User, error)
	CreateUser(ctx context.Context, in identity.NewUser) (*identity.User, string, error)
	UpdateUser(ctx context.Context, userID string, in identity.UserPatch) (*identity.User, error)
	ResetPassword(ctx context.Context, userID string) (*identity.User, string, error)
	DeleteUser(ctx context.Context, actorID, userID string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) (*identity.User, error)
	UpdateProfile(ctx context.Context, userID string, in identity.ProfilePatch) (*identity.User, error)
	Authenticate(ctx context.Context, username, password string) (*identity.User, error)
}

// Provider is the internal OIDC IdP (authorization code + PKCE).
type Provider struct {
	cfg      Config
	ident    identityService
	key      *rsaHolder
	requests *ttlcache.Cache[string, authRequest]
	codes    *ttlcache.Cache[string, authCode]
	now      func() time.Time
}

type rsaHolder struct {
	kid    string
	sign   func(claims map[string]any) (string, error)
	verify func(token string) (map[string]any, error)
	jwks   map[string]any
}

// New constructs the IdP, loading or creating the RSA signing key.
func New(cfg Config, ident identityService) (*Provider, error) {
	if ident == nil {
		return nil, fmt.Errorf("identity service is required")
	}
	if cfg.ClientID == "" {
		cfg.ClientID = DefaultClientID
	}
	if cfg.Audience == "" {
		cfg.Audience = cfg.ClientID
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "https://localhost/auth"
	}
	if cfg.LoginPath == "" {
		cfg.LoginPath = DefaultLoginPath
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = time.Hour
	}
	if cfg.IDTokenTTL <= 0 {
		cfg.IDTokenTTL = time.Hour
	}
	if cfg.AuthCodeTTL <= 0 {
		cfg.AuthCodeTTL = 10 * time.Minute
	}
	if cfg.AuthRequestTTL <= 0 {
		cfg.AuthRequestTTL = 10 * time.Minute
	}

	key, err := loadOrCreateKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("idp signing key: %w", err)
	}
	kid := keyID(&key.PublicKey)
	h := &rsaHolder{
		kid:  kid,
		jwks: jwksFor(&key.PublicKey, kid),
		sign: func(claims map[string]any) (string, error) {
			return signRS256(key, kid, claims)
		},
		verify: func(token string) (map[string]any, error) {
			return verifyRS256(&key.PublicKey, token)
		},
	}

	return &Provider{
		cfg:      cfg,
		ident:    ident,
		key:      h,
		requests: ttlcache.New(ttlcache.WithTTL[string, authRequest](cfg.AuthRequestTTL), ttlcache.WithAutoDeletion[string, authRequest](time.Minute)),
		codes:    ttlcache.New(ttlcache.WithTTL[string, authCode](cfg.AuthCodeTTL), ttlcache.WithAutoDeletion[string, authCode](time.Minute)),
		now:      time.Now,
	}, nil
}

// Shutdown implements do.Shutdowner.
func (p *Provider) Shutdown() error {
	if p.requests != nil {
		p.requests.Close()
	}
	if p.codes != nil {
		p.codes.Close()
	}
	return nil
}

// Config returns a copy of the IdP config.
func (p *Provider) Config() Config {
	return p.cfg
}

// CookieName is the login-session cookie.
func CookieName() string {
	return cookieName
}

// Discovery is the OIDC discovery document.
func (p *Provider) Discovery() map[string]any {
	iss := p.cfg.Issuer
	return map[string]any{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/authorize",
		"token_endpoint":                        iss + "/token",
		"userinfo_endpoint":                     iss + "/userinfo",
		"jwks_uri":                              iss + "/jwks",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"code_challenge_methods_supported":      []string{pkceS256},
		"scopes_supported":                      []string{"openid", "profile"},
		"claims_supported":                      []string{"sub", "preferred_username", "roles", "iss", "aud", "exp", "iat"},
	}
}

// JWKS returns the public signing key set.
func (p *Provider) JWKS() map[string]any {
	return p.key.jwks
}

// StartAuthorization validates an authorize request and returns a session id plus the login path.
func (p *Provider) StartAuthorization(req AuthorizationRequest) (requestID string, loginPath string, err error) {
	if req.ResponseType != "code" {
		return "", "", fmt.Errorf("%w: response_type must be code", ErrInvalidRequest)
	}
	if req.ClientID != p.cfg.ClientID {
		return "", "", fmt.Errorf("%w: unknown client_id", ErrInvalidClient)
	}
	if !p.cfg.AllowsRedirect(req.RedirectURI) {
		logger.Warn(fmt.Sprintf("rejected redirect_uri=%q allowed=%v", req.RedirectURI, p.cfg.RedirectURIs))
		return "", "", fmt.Errorf("%w: redirect_uri %q is not registered", ErrInvalidRequest, req.RedirectURI)
	}
	if !hasScope(req.Scope, "openid") {
		return "", "", fmt.Errorf("%w: scope must include openid", ErrInvalidRequest)
	}
	if req.CodeChallengeMethod != pkceS256 {
		return "", "", fmt.Errorf("%w: code_challenge_method must be S256", ErrInvalidRequest)
	}
	if req.CodeChallenge == "" {
		return "", "", fmt.Errorf("%w: code_challenge is required", ErrInvalidRequest)
	}

	id, err := randomURLToken(32)
	if err != nil {
		return "", "", err
	}
	p.requests.Add(id, authRequest{
		ID:                  id,
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	})
	return id, p.cfg.LoginPath, nil
}

// CompleteLogin authenticates the user for a pending authorize request.
func (p *Provider) CompleteLogin(ctx context.Context, requestID, username, password string) (LoginResult, error) {
	ar, err := p.getRequest(requestID)
	if err != nil {
		return LoginResult{}, err
	}
	u, err := p.ident.Authenticate(ctx, username, password)
	if err != nil {
		return LoginResult{}, err
	}
	ar.UserID = u.ID
	ar.PasswordChangedOK = !u.MustChangePassword
	p.requests.Add(requestID, ar)
	if u.MustChangePassword {
		return LoginResult{Status: LoginPasswordChange}, nil
	}
	return p.issueCode(ar)
}

// CompletePasswordChange finishes mustChangePassword and then issues the auth code.
func (p *Provider) CompletePasswordChange(ctx context.Context, requestID, oldPassword, newPassword string) (LoginResult, error) {
	ar, err := p.getRequest(requestID)
	if err != nil {
		return LoginResult{}, err
	}
	if ar.UserID == "" {
		return LoginResult{}, ErrLoginRequired
	}
	u, err := p.ident.ChangePassword(ctx, ar.UserID, oldPassword, newPassword)
	if err != nil {
		return LoginResult{}, err
	}
	ar.UserID = u.ID
	ar.PasswordChangedOK = true
	p.requests.Add(requestID, ar)
	return p.issueCode(ar)
}

func (p *Provider) issueCode(ar authRequest) (LoginResult, error) {
	if ar.UserID == "" || !ar.PasswordChangedOK {
		return LoginResult{}, ErrPasswordChange
	}
	code, err := randomURLToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	p.codes.Add(code, authCode{Code: code, Request: ar, ExpiresAt: p.now().Add(p.cfg.AuthCodeTTL)})
	p.requests.Delete(ar.ID)

	u, err := url.Parse(ar.RedirectURI)
	if err != nil {
		return LoginResult{}, fmt.Errorf("%w: redirect_uri", ErrInvalidRequest)
	}
	q := u.Query()
	q.Set("code", code)
	if ar.State != "" {
		q.Set("state", ar.State)
	}
	u.RawQuery = q.Encode()
	return LoginResult{Status: LoginOK, RedirectTo: u.String()}, nil
}

// ExchangeToken implements authorization_code + PKCE.
func (p *Provider) ExchangeToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	if req.GrantType != "authorization_code" {
		return nil, ErrUnsupportedGrant
	}
	if req.ClientID != p.cfg.ClientID {
		return nil, ErrInvalidClient
	}
	ac, ok := p.codes.Get(req.Code)
	if !ok {
		return nil, ErrInvalidGrant
	}
	if p.now().After(ac.ExpiresAt) {
		p.codes.Delete(req.Code)
		return nil, ErrInvalidGrant
	}
	if ac.Request.RedirectURI != req.RedirectURI {
		return nil, ErrInvalidGrant
	}
	if err := verifyPKCE(req.CodeVerifier, ac.Request.CodeChallenge, ac.Request.CodeChallengeMethod); err != nil {
		return nil, ErrInvalidGrant
	}
	p.codes.Delete(req.Code)
	u, err := p.ident.GetByID(ctx, ac.Request.UserID)
	if err != nil {
		return nil, ErrInvalidGrant
	}
	if u.MustChangePassword {
		return nil, ErrPasswordChange
	}
	return p.mintTokens(u, ac.Request)
}

func (p *Provider) mintTokens(u *identity.User, ar authRequest) (*TokenResponse, error) {
	now := p.now().UTC()
	accessExp := now.Add(p.cfg.AccessTokenTTL)
	idExp := now.Add(p.cfg.IDTokenTTL)

	accessClaims := map[string]any{
		"iss":                p.cfg.Issuer,
		"sub":                u.ID,
		"aud":                p.cfg.Audience,
		"iat":                unixTime(now),
		"exp":                unixTime(accessExp),
		"token_use":          "access",
		"scope":              ar.Scope,
		"preferred_username": u.Username,
		"roles":              u.Roles,
		"mustChangePassword": false,
	}
	idClaims := map[string]any{
		"iss":                p.cfg.Issuer,
		"sub":                u.ID,
		"aud":                ar.ClientID,
		"iat":                unixTime(now),
		"exp":                unixTime(idExp),
		"token_use":          "id",
		"preferred_username": u.Username,
		"roles":              u.Roles,
	}
	if ar.Nonce != "" {
		idClaims["nonce"] = ar.Nonce
	}

	access, err := p.key.sign(accessClaims)
	if err != nil {
		return nil, err
	}
	idToken, err := p.key.sign(idClaims)
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(p.cfg.AccessTokenTTL / time.Second),
		IDToken:     idToken,
		Scope:       ar.Scope,
	}, nil
}

// UserInfoFromAccessToken validates an access token and returns userinfo.
func (p *Provider) UserInfoFromAccessToken(ctx context.Context, token string) (*UserInfo, error) {
	claims, err := p.VerifyAccessToken(token)
	if err != nil {
		return nil, err
	}
	u, err := p.ident.GetByID(ctx, claimString(claims, "sub"))
	if err != nil {
		return nil, ErrInvalidToken
	}
	return &UserInfo{
		Subject:            u.ID,
		PreferredUsername:  u.Username,
		Roles:              u.Roles,
		MustChangePassword: u.MustChangePassword,
	}, nil
}

// VerifyAccessToken checks signature, issuer, audience, expiry and token_use.
func (p *Provider) VerifyAccessToken(token string) (map[string]any, error) {
	if strings.HasPrefix(strings.ToUpper(token), "BEARER ") {
		token = strings.TrimSpace(token[7:])
	}
	claims, err := p.key.verify(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claimString(claims, "iss") != p.cfg.Issuer {
		return nil, ErrInvalidToken
	}
	if !audienceMatches(claims["aud"], p.cfg.Audience) {
		return nil, ErrInvalidToken
	}
	if use := claimString(claims, "token_use"); use != "" && use != "access" {
		return nil, ErrInvalidToken
	}
	if exp, ok := claimTime(claims["exp"]); ok && p.now().After(exp) {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (p *Provider) getRequest(id string) (authRequest, error) {
	if id == "" {
		return authRequest{}, ErrLoginRequired
	}
	ar, ok := p.requests.Get(id)
	if !ok {
		return authRequest{}, ErrLoginRequired
	}
	return *ar, nil
}

func hasScope(scope, want string) bool {
	for _, s := range strings.Fields(scope) {
		if s == want {
			return true
		}
	}
	return false
}

func audienceMatches(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
