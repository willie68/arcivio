package idp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/arcivio/internal/domain/idp"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

const cookieMaxAge = 600

// Handler exposes the internal OIDC IdP under /auth.
type Handler struct {
	idp *idp.Provider
}

// New creates the HTTP adapter for the IdP.
func New(p *idp.Provider) *Handler {
	return &Handler{idp: p}
}

// Routes implements api.Handler.
func (h *Handler) Routes() (string, *chi.Mux) {
	r := chi.NewRouter()
	r.Get("/.well-known/openid-configuration", h.Discovery)
	r.Get("/jwks", h.JWKS)
	r.Get("/authorize", h.Authorize)
	r.Post("/login", h.Login)
	r.Post("/change-password", h.ChangePassword)
	r.Post("/token", h.Token)
	r.Get("/userinfo", h.Userinfo)
	r.Post("/logout", h.Logout)
	return "/auth", r
}

func (h *Handler) Discovery(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, h.idp.Discovery())
}

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, h.idp.JWKS())
}

func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := idp.AuthorizationRequest{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}
	requestID, loginPath, err := h.idp.StartAuthorization(req)
	if err != nil {
		writeOIDCError(w, r, err)
		return
	}
	h.setSessionCookie(w, r, requestID)
	http.Redirect(w, r, loginPath, http.StatusFound)
}

type loginBody struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	res, err := h.idp.CompleteLogin(r.Context(), h.sessionID(r), body.Username, body.Password)
	if err != nil {
		writeLoginError(w, r, err)
		return
	}
	render.JSON(w, r, res)
}

type changePasswordBody struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var body changePasswordBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	res, err := h.idp.CompletePasswordChange(r.Context(), h.sessionID(r), body.OldPassword, body.NewPassword)
	if err != nil {
		writeLoginError(w, r, err)
		return
	}
	render.JSON(w, r, res)
}

func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, r, idp.ErrInvalidRequest)
		return
	}
	tok, err := h.idp.ExchangeToken(r.Context(), idp.TokenRequest{
		GrantType:    r.PostForm.Get("grant_type"),
		Code:         r.PostForm.Get("code"),
		RedirectURI:  r.PostForm.Get("redirect_uri"),
		ClientID:     r.PostForm.Get("client_id"),
		CodeVerifier: r.PostForm.Get("code_verifier"),
	})
	if err != nil {
		writeOIDCError(w, r, err)
		return
	}
	render.JSON(w, r, tok)
}

func (h *Handler) Userinfo(w http.ResponseWriter, r *http.Request) {
	token := bearer(r)
	if token == "" {
		writeBearerError(w, "missing access token", http.StatusUnauthorized)
		return
	}
	info, err := h.idp.UserInfoFromAccessToken(r.Context(), token)
	if err != nil {
		writeBearerError(w, "invalid access token", http.StatusUnauthorized)
		return
	}
	render.JSON(w, r, info)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     idp.CookieName(),
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sessionID(r *http.Request) string {
	c, err := r.Cookie(idp.CookieName())
	if err != nil {
		return ""
	}
	return c.Value
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     idp.CookieName(),
		Value:    id,
		Path:     "/auth",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func cookieSecure(r *http.Request) bool {
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		proto := strings.TrimSpace(strings.Split(xf, ",")[0])
		return strings.EqualFold(proto, "https")
	}
	return r.TLS != nil
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func writeLoginError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, identity.ErrInvalidCredentials):
		httputils.Err(w, r, serror.Unauthorized(err, "invalid-credentials", "invalid username or password"))
	case errors.Is(err, identity.ErrWeakPassword), errors.Is(err, identity.ErrSamePassword):
		httputils.Err(w, r, serror.BadRequest(err, "invalid-password", err.Error()))
	case errors.Is(err, idp.ErrLoginRequired):
		httputils.Err(w, r, serror.Unauthorized(err, "login-required", "start the authorization code flow first"))
	case errors.Is(err, idp.ErrPasswordChange):
		httputils.Err(w, r, serror.Unauthorized(err, "password-change-required", "password must be changed"))
	default:
		httputils.Err(w, r, err)
	}
}

func writeOIDCError(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusBadRequest
	key := "invalid_request"
	switch {
	case errors.Is(err, idp.ErrInvalidClient), errors.Is(err, idp.ErrUnauthorizedClient):
		key = "invalid_client"
		code = http.StatusUnauthorized
	case errors.Is(err, idp.ErrInvalidGrant):
		key = "invalid_grant"
	case errors.Is(err, idp.ErrUnsupportedGrant):
		key = "unsupported_grant_type"
	case errors.Is(err, idp.ErrInvalidToken):
		key = "invalid_token"
		code = http.StatusUnauthorized
	case errors.Is(err, idp.ErrInvalidRequest):
		key = "invalid_request"
	}
	render.Status(r, code)
	render.JSON(w, r, map[string]string{
		"error":             key,
		"error_description": err.Error(),
	})
}

func writeBearerError(w http.ResponseWriter, desc string, status int) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             "invalid_token",
		"error_description": desc,
	})
}
