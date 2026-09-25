package idp

import (
	"errors"
	"time"
)

// Protocol errors (mapped to OIDC / HTTP by the inbound adapter).
var (
	ErrInvalidRequest     = errors.New("invalid_request")
	ErrInvalidClient      = errors.New("invalid_client")
	ErrInvalidGrant       = errors.New("invalid_grant")
	ErrUnsupportedGrant   = errors.New("unsupported_grant_type")
	ErrLoginRequired      = errors.New("login_required")
	ErrPasswordChange     = errors.New("password_change_required")
	ErrUnauthorizedClient = errors.New("unauthorized_client")
	ErrInvalidToken       = errors.New("invalid_token")
)

// AuthorizationRequest is an OIDC authorize query (code + PKCE).
type AuthorizationRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// LoginResult is returned after username/password (and optional password change).
type LoginResult struct {
	Status     string `json:"status"`
	RedirectTo string `json:"redirectTo,omitempty"`
}

const (
	LoginOK             = "ok"
	LoginPasswordChange = "password_change_required"
)

// TokenRequest is an OIDC token request (authorization_code + PKCE).
type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	CodeVerifier string
}

// TokenResponse is a successful token endpoint payload.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope,omitempty"`
}

// UserInfo is the OIDC userinfo payload.
type UserInfo struct {
	Subject            string   `json:"sub"`
	PreferredUsername  string   `json:"preferred_username"`
	Roles              []string `json:"roles"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

type authRequest struct {
	ID                  string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              string
	PasswordChangedOK   bool
}

type authCode struct {
	Code      string
	Request   authRequest
	ExpiresAt time.Time
}
