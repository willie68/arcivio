package roles

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/roles"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// TextResponse is a short UI string in German and English.
type TextResponse struct {
	De string `json:"de"`
	En string `json:"en"`
}

// RoleResponse is one built-in role for the settings page.
type RoleResponse struct {
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
}

// RoleListResponse is the built-in role catalog.
type RoleListResponse struct {
	Items []RoleResponse `json:"items"`
}

type rolesService interface {
	GetRoles() []roles.RoleDefinition
	HasRole(checkRole string, desiredRole string) bool
}

// Handler serves the role catalog under /roles.
type Handler struct {
	roles rolesService
}

// New creates the HTTP adapter for the built-in roles.
func New(inj do.Injector) *Handler {
	return &Handler{roles: do.MustInvokeAs[rolesService](inj)}
}

// Routes implements api.Handler.
func (h *Handler) Routes() (string, *chi.Mux) {
	r := chi.NewRouter()
	r.Get("/", h.List)
	return "/roles", r
}

// List godoc
//
//	@Summary		List roles
//	@Description	Returns the built-in roles with labels and descriptions. Requires the admin role.
//	@Tags			identity
//	@Produce		json
//	@Success		200	{object}	RoleListResponse
//	@Failure		401
//	@Failure		403
//	@Router			/roles [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	listed := h.roles.GetRoles()
	items := make([]RoleResponse, 0, len(listed))
	for _, role := range listed {
		items = append(items, RoleResponse{
			Name:        role.Name,
			Labels:      TextResponse{De: role.Labels.De, En: role.Labels.En},
			Description: TextResponse{De: role.Description.De, En: role.Description.En},
		})
	}
	render.JSON(w, r, RoleListResponse{Items: items})
}

func requireAdmin(roleService rolesService, w http.ResponseWriter, r *http.Request) bool {
	token, claims, err := auth.FromContext(r.Context())
	if err != nil || token == nil || !token.IsValid {
		httputils.Err(w, r, serror.Unauthorized(err, "unauthorized", "login required"))
		return false
	}
	for _, name := range claimRoles(claims) {
		if roleService.HasRole(name, roles.RoleAdmin) {
			return true
		}
	}
	httputils.Err(w, r, serror.Forbidden(errors.New("admin role required"), "forbidden", "admin role required"))
	return false
}

func claimRoles(claims map[string]any) []string {
	switch raw := claims["roles"].(type) {
	case []string:
		return raw
	case []any:
		names := make([]string, 0, len(raw))
		for _, item := range raw {
			name, ok := item.(string)
			if ok {
				names = append(names, name)
			}
		}
		return names
	default:
		return nil
	}
}
