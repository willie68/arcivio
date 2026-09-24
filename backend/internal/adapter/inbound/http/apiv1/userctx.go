package apiv1

import (
	"errors"
	"net/http"

	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

func currentUser(ident *identity.Service, w http.ResponseWriter, r *http.Request) (*identity.User, bool) {
	token, claims, err := auth.FromContext(r.Context())
	if err != nil || token == nil || !token.IsValid {
		httputils.Err(w, r, serror.Unauthorized(err, "unauthorized", "login required"))
		return nil, false
	}
	sub, _ := claims["sub"].(string)
	u, err := ident.GetByID(r.Context(), sub)
	if err != nil {
		httputils.Err(w, r, serror.Unauthorized(err, "unauthorized", "unknown user"))
		return nil, false
	}
	return u, true
}

func requireAdmin(ident *identity.Service, w http.ResponseWriter, r *http.Request) (*identity.User, bool) {
	u, ok := currentUser(ident, w, r)
	if !ok {
		return nil, false
	}
	if !identity.HasRole(u, identity.RoleAdmin) {
		httputils.Err(w, r, serror.Forbidden(errors.New("admin role required"), "forbidden", "admin role required"))
		return nil, false
	}
	return u, true
}
