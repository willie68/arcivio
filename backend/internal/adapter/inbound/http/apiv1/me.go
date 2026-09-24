package apiv1

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// MeResponse is the current authenticated user.
type MeResponse struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	Email              string     `json:"email"`
	Roles              []string   `json:"roles"`
	MustChangePassword bool       `json:"mustChangePassword"`
	LastLogin          *time.Time `json:"lastLogin"`
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
	u, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	render.JSON(w, r, MeResponse{
		ID:                 u.ID,
		Username:           u.Username,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		Email:              u.Email,
		Roles:              u.Roles,
		MustChangePassword: u.MustChangePassword,
		LastLogin:          u.LastLogin,
	})
}

type changePasswordBody struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

// ChangePassword godoc
//
//	@Summary		Change own password
//	@Description	Changes the password of the authenticated user.
//	@Tags			identity
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		400
//	@Failure		401
//	@Router			/me/password [post]
func (h *meHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var body changePasswordBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	if _, err := h.ident.ChangePassword(r.Context(), u.ID, body.OldPassword, body.NewPassword); err != nil {
		switch {
		case errors.Is(err, identity.ErrInvalidCredentials):
			httputils.Err(w, r, serror.BadRequest(err, "invalid-password", "old password is wrong"))
		case errors.Is(err, identity.ErrWeakPassword), errors.Is(err, identity.ErrSamePassword):
			httputils.Err(w, r, serror.BadRequest(err, "invalid-password", err.Error()))
		default:
			httputils.Err(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *meHandler) currentUser(w http.ResponseWriter, r *http.Request) (*identity.User, bool) {
	return currentUser(h.ident, w, r)
}
