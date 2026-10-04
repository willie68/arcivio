package fieldgroups

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/roles"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// TextResponse is a short UI string in German and English.
type TextResponse struct {
	De string `json:"de"`
	En string `json:"en"`
}

// FieldResponse is one field definition.
type FieldResponse struct {
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
	ValueType   string       `json:"valueType"`
}

// GroupResponse is one field group definition.
type GroupResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Labels      TextResponse    `json:"labels"`
	Description TextResponse    `json:"description"`
	Fields      []FieldResponse `json:"fields"`
}

// GroupListResponse is the field-group catalog.
type GroupListResponse struct {
	Items []GroupResponse `json:"items"`
}

type groupBody struct {
	Name        string          `json:"name"`
	Labels      TextResponse    `json:"labels"`
	Description TextResponse    `json:"description"`
	Fields      []FieldResponse `json:"fields"`
}

type groupService interface {
	List(ctx context.Context) ([]fieldgroup.Group, error)
	Get(ctx context.Context, id string) (*fieldgroup.Group, error)
	Create(ctx context.Context, in fieldgroup.Input) (*fieldgroup.Group, error)
	Update(ctx context.Context, id string, in fieldgroup.Input) (*fieldgroup.Group, error)
	Delete(ctx context.Context, id string) error
}

type rolesService interface {
	HasRole(checkRole string, desiredRole string) bool
}

// Handler serves field group definitions under /field-groups.
type Handler struct {
	groups groupService
	roles  rolesService
}

// New creates the HTTP adapter for field groups.
func New(inj do.Injector) *Handler {
	return &Handler{
		groups: do.MustInvokeAs[groupService](inj),
		roles:  do.MustInvokeAs[rolesService](inj),
	}
}

// Routes implements api.Handler.
func (h *Handler) Routes() (string, *chi.Mux) {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return "/field-groups", r
}

// List godoc
//
//	@Summary		List field groups
//	@Description	Returns field group definitions. Requires the admin role.
//	@Tags			field-groups
//	@Produce		json
//	@Success		200	{object}	GroupListResponse
//	@Failure		401
//	@Failure		403
//	@Router			/field-groups [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	listed, err := h.groups.List(r.Context())
	if err != nil {
		httputils.Err(w, r, err)
		return
	}
	items := make([]GroupResponse, 0, len(listed))
	for _, group := range listed {
		items = append(items, toResponse(group))
	}
	render.JSON(w, r, GroupListResponse{Items: items})
}

// Get godoc
//
//	@Summary		Get field group
//	@Description	Returns one field group definition. Requires the admin role.
//	@Tags			field-groups
//	@Produce		json
//	@Param			id	path		string	true	"field group id"
//	@Success		200	{object}	GroupResponse
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Router			/field-groups/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	group, err := h.groups.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeGroupError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*group))
}

// Create godoc
//
//	@Summary		Create field group
//	@Description	Stores a field group definition. Requires the admin role.
//	@Tags			field-groups
//	@Accept			json
//	@Produce		json
//	@Success		201	{object}	GroupResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		409
//	@Router			/field-groups [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body groupBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	group, err := h.groups.Create(r.Context(), toInput(body))
	if err != nil {
		writeGroupError(w, r, err)
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toResponse(*group))
}

// Update godoc
//
//	@Summary		Update field group
//	@Description	Replaces name, texts and fields. Requires the admin role.
//	@Tags			field-groups
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"field group id"
//	@Success		200	{object}	GroupResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Failure		409
//	@Router			/field-groups/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body groupBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	group, err := h.groups.Update(r.Context(), chi.URLParam(r, "id"), toInput(body))
	if err != nil {
		writeGroupError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*group))
}

// Delete godoc
//
//	@Summary		Delete field group
//	@Description	Deletes a field group definition. Requires the admin role.
//	@Tags			field-groups
//	@Param			id	path	string	true	"field group id"
//	@Success		204
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Router			/field-groups/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	if err := h.groups.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeGroupError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toInput(body groupBody) fieldgroup.Input {
	fields := make([]fieldgroup.Field, 0, len(body.Fields))
	for _, field := range body.Fields {
		fields = append(fields, fieldgroup.Field{
			Name:        field.Name,
			Labels:      fieldgroup.Text{De: field.Labels.De, En: field.Labels.En},
			Description: fieldgroup.Text{De: field.Description.De, En: field.Description.En},
			ValueType:   field.ValueType,
		})
	}
	return fieldgroup.Input{
		Name:        body.Name,
		Labels:      fieldgroup.Text{De: body.Labels.De, En: body.Labels.En},
		Description: fieldgroup.Text{De: body.Description.De, En: body.Description.En},
		Fields:      fields,
	}
}

func toResponse(group fieldgroup.Group) GroupResponse {
	fields := make([]FieldResponse, 0, len(group.Fields))
	for _, field := range group.Fields {
		fields = append(fields, FieldResponse{
			Name:        field.Name,
			Labels:      TextResponse{De: field.Labels.De, En: field.Labels.En},
			Description: TextResponse{De: field.Description.De, En: field.Description.En},
			ValueType:   field.ValueType,
		})
	}
	return GroupResponse{
		ID:          group.ID,
		Name:        group.Name,
		Labels:      TextResponse{De: group.Labels.De, En: group.Labels.En},
		Description: TextResponse{De: group.Description.De, En: group.Description.En},
		Fields:      fields,
	}
}

func writeGroupError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, fieldgroup.ErrAlreadyExists):
		httputils.Err(w, r, serror.New(http.StatusConflict, "already-exists", "field group already exists"))
	case errors.Is(err, fieldgroup.ErrNotFound):
		httputils.Err(w, r, serror.NotFound("field-group", "", err))
	case errors.Is(err, fieldgroup.ErrInvalid):
		httputils.Err(w, r, serror.BadRequest(err, "invalid-field-group", "invalid field group"))
	default:
		httputils.Err(w, r, err)
	}
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
