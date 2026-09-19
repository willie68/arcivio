package apiv1

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// MeResponse is the current authenticated user.
type MeResponse struct {
	ID                 string   `json:"id"`
	Username           string   `json:"username"`
	Roles              []string `json:"roles"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

type meHandler struct {
	ident *identity.Service
}

func newMeHandler(inj do.Injector) *meHandler {
	return &meHandler{ident: do.MustInvoke[*identity.Service](inj)}
}

// GetMe godoc
//
//	@Summary		Current user
//	@Description	Returns the authenticated local user from the access token.
//	@Tags			identity
//	@Produce		json
//	@Success		200	{object}	MeResponse
//	@Failure		401
//	@Router			/me [get]
func (h *meHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	token, claims, err := auth.FromContext(r.Context())
	if err != nil || token == nil || !token.IsValid {
		httputils.Err(w, r, serror.Unauthorized(err, "unauthorized", "login required"))
		return
	}
	sub, _ := claims["sub"].(string)
	u, err := h.ident.GetByID(r.Context(), sub)
	if err != nil {
		httputils.Err(w, r, serror.Unauthorized(err, "unauthorized", "unknown user"))
		return
	}
	render.JSON(w, r, MeResponse{
		ID:                 u.ID,
		Username:           u.Username,
		Roles:              u.Roles,
		MustChangePassword: u.MustChangePassword,
	})
}
