package users

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/identity"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// UserResponse is a local user without secrets.
type UserResponse struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	Email              string     `json:"email"`
	Roles              []string   `json:"roles"`
	MustChangePassword bool       `json:"mustChangePassword"`
	LastLogin          *time.Time `json:"lastLogin"`
}

// UserListResponse is one page of users.
type UserListResponse struct {
	Items    []UserResponse `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type identityService interface {
	ListUsers(ctx context.Context, offset, limit int, sort string, desc bool, prefix string) ([]identity.User, int, error)
	GetByID(ctx context.Context, id string) (*identity.User, error)
	CreateUser(ctx context.Context, in identity.NewUser) (*identity.User, string, error)
	UpdateUser(ctx context.Context, userID string, in identity.UserPatch) (*identity.User, error)
	ResetPassword(ctx context.Context, userID string) (*identity.User, string, error)
	DeleteUser(ctx context.Context, actorID, userID string) error
}

// Handler serves the admin user API under /users.
type Handler struct {
	ident identityService
}

// New creates the HTTP adapter for local users.
func New(inj do.Injector) *Handler {
	return &Handler{ident: do.MustInvokeAs[identityService](inj)}
}

// Routes implements api.Handler.
func (h *Handler) Routes() (string, *chi.Mux) {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Post("/{id}/password-reset", h.ResetPassword)
	r.Delete("/{id}", h.Delete)
	return "/users", r
}

// List godoc
//
//	@Summary		List users
//	@Description	Returns a page of local users. Requires the admin role.
//	@Tags			identity
//	@Produce		json
//	@Param			page		query		int		false	"1-based page"
//	@Param			pageSize	query		int		false	"page size, max 100"
//	@Param			sort		query		string	false	"username, firstName, lastName, email, roles, lastLogin or mustChangePassword"
//	@Param			order		query		string	false	"asc or desc"
//	@Param			prefix		query		string	false	"prefix on login name, first name, last name or email; applied from 3 characters"
//	@Success		200			{object}	UserListResponse
//	@Failure		401
//	@Failure		403
//	@Router			/users [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(h.ident, w, r); !ok {
		return
	}
	page := queryPositive(r, "page", 1)
	pageSize := queryPositive(r, "pageSize", 10)
	if pageSize > 100 {
		pageSize = 100
	}
	sortField, desc := querySort(r)
	listed, total, err := h.ident.ListUsers(r.Context(), (page-1)*pageSize, pageSize, sortField, desc, r.URL.Query().Get("prefix"))
	if err != nil {
		httputils.Err(w, r, err)
		return
	}
	items := make([]UserResponse, 0, len(listed))
	for _, u := range listed {
		items = append(items, toUserResponse(u))
	}
	render.JSON(w, r, UserListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

type createUserBody struct {
	Username  string   `json:"username"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
}

type createUserResponse struct {
	User     UserResponse `json:"user"`
	Password string       `json:"password"`
}

// Create godoc
//
//	@Summary		Create user
//	@Description	Creates a local user and returns a one-time password. Requires the admin role.
//	@Tags			identity
//	@Accept			json
//	@Produce		json
//	@Success		201	{object}	createUserResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		409
//	@Router			/users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(h.ident, w, r); !ok {
		return
	}
	var body createUserBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	u, password, err := h.ident.CreateUser(r.Context(), identity.NewUser{
		Username:  body.Username,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Roles:     body.Roles,
	})
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, createUserResponse{User: toUserResponse(*u), Password: password})
}

// Update godoc
//
//	@Summary		Update user
//	@Description	Changes login name, profile and roles. The password is unchanged. Requires the admin role.
//	@Tags			identity
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string			true	"user id"
//	@Success		200	{object}	UserResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Failure		409
//	@Router			/users/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(h.ident, w, r); !ok {
		return
	}
	var body createUserBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	u, err := h.ident.UpdateUser(r.Context(), chi.URLParam(r, "id"), identity.UserPatch{
		Username:  body.Username,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Roles:     body.Roles,
	})
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	render.JSON(w, r, toUserResponse(*u))
}

// ResetPassword godoc
//
//	@Summary		Reset user password
//	@Description	Sets a new one-time password and requires a change at the next login. Requires the admin role.
//	@Tags			identity
//	@Produce		json
//	@Param			id	path		string	true	"user id"
//	@Success		200	{object}	createUserResponse
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Router			/users/{id}/password-reset [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(h.ident, w, r); !ok {
		return
	}
	u, password, err := h.ident.ResetPassword(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	render.JSON(w, r, createUserResponse{User: toUserResponse(*u), Password: password})
}

// Delete godoc
//
//	@Summary		Delete user
//	@Description	Deletes a local user. Requires the admin role. The signed-in user and the last admin stay.
//	@Tags			identity
//	@Param			id	path	string	true	"user id"
//	@Success		204
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Failure		409
//	@Router			/users/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdmin(h.ident, w, r)
	if !ok {
		return
	}
	if err := h.ident.DeleteUser(r.Context(), actor.ID, chi.URLParam(r, "id")); err != nil {
		writeUserError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toUserResponse(u identity.User) UserResponse {
	return UserResponse{
		ID:                 u.ID,
		Username:           u.Username,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		Email:              u.Email,
		Roles:              u.Roles,
		MustChangePassword: u.MustChangePassword,
		LastLogin:          u.LastLogin,
	}
}

func writeUserError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, identity.ErrAlreadyExists):
		httputils.Err(w, r, serror.New(http.StatusConflict, "already-exists", "user already exists"))
	case errors.Is(err, identity.ErrUserNotFound):
		httputils.Err(w, r, serror.NotFound("user", "", err))
	case errors.Is(err, identity.ErrDeleteSelf):
		httputils.Err(w, r, serror.New(http.StatusConflict, "delete-self", "cannot delete the signed-in user"))
	case errors.Is(err, identity.ErrLastAdmin):
		httputils.Err(w, r, serror.New(http.StatusConflict, "last-admin", "cannot delete the last admin"))
	case errors.Is(err, identity.ErrInvalidRole), errors.Is(err, identity.ErrInvalidEmail):
		httputils.Err(w, r, serror.BadRequest(err, "invalid-user", err.Error()))
	default:
		if strings.Contains(err.Error(), "username required") {
			httputils.Err(w, r, serror.BadRequest(err, "invalid-user", err.Error()))
			return
		}
		httputils.Err(w, r, err)
	}
}

func querySort(r *http.Request) (string, bool) {
	order := strings.ToLower(r.URL.Query().Get("order"))
	return r.URL.Query().Get("sort"), order == "desc"
}

func queryPositive(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func currentUser(ident identityService, w http.ResponseWriter, r *http.Request) (*identity.User, bool) {
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

func requireAdmin(ident identityService, w http.ResponseWriter, r *http.Request) (*identity.User, bool) {
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
