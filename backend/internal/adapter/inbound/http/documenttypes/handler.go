package documenttypes

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/documenttype"
	"github.com/willie68/arcivio/internal/domain/roles"
	"github.com/willie68/arcivio/internal/shared/serror"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
)

// TextResponse is a short UI string in German and English.
type TextResponse struct {
	De string `json:"de"`
	En string `json:"en"`
}

// TypeResponse is one document type definition.
type TypeResponse struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Labels        TextResponse `json:"labels"`
	Description   TextResponse `json:"description"`
	FieldGroupIDs []string     `json:"fieldGroupIds"`
}

// TypeListResponse is the document-type catalog.
type TypeListResponse struct {
	Items []TypeResponse `json:"items"`
}

type typeBody struct {
	Name          string       `json:"name"`
	Labels        TextResponse `json:"labels"`
	Description   TextResponse `json:"description"`
	FieldGroupIDs []string     `json:"fieldGroupIds"`
}

type typeService interface {
	List(ctx context.Context) ([]documenttype.Type, error)
	Get(ctx context.Context, id string) (*documenttype.Type, error)
	Create(ctx context.Context, in documenttype.Input) (*documenttype.Type, error)
	Update(ctx context.Context, id string, in documenttype.Input) (*documenttype.Type, error)
	Delete(ctx context.Context, id string) error
}

type rolesService interface {
	HasRole(checkRole string, desiredRole string) bool
}

// Handler serves document type definitions under /document-types.
type Handler struct {
	types typeService
	roles rolesService
}

// New creates the HTTP adapter for document types.
func New(inj do.Injector) *Handler {
	return &Handler{
		types: do.MustInvokeAs[typeService](inj),
		roles: do.MustInvokeAs[rolesService](inj),
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
	return "/document-types", r
}

// List godoc
//
//	@Summary		List document types
//	@Description	Returns document type definitions. Requires the admin role.
//	@Tags			document-types
//	@Produce		json
//	@Success		200	{object}	TypeListResponse
//	@Failure		401
//	@Failure		403
//	@Router			/document-types [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	listed, err := h.types.List(r.Context())
	if err != nil {
		httputils.Err(w, r, err)
		return
	}
	items := make([]TypeResponse, 0, len(listed))
	for _, item := range listed {
		items = append(items, toResponse(item))
	}
	render.JSON(w, r, TypeListResponse{Items: items})
}

// Get godoc
//
//	@Summary		Get document type
//	@Description	Returns one document type definition. Requires the admin role.
//	@Tags			document-types
//	@Produce		json
//	@Param			id	path		string	true	"document type id"
//	@Success		200	{object}	TypeResponse
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Router			/document-types/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	item, err := h.types.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*item))
}

// Create godoc
//
//	@Summary		Create document type
//	@Description	Stores a document type definition. Requires the admin role.
//	@Tags			document-types
//	@Accept			json
//	@Produce		json
//	@Success		201	{object}	TypeResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		409
//	@Router			/document-types [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body typeBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	item, err := h.types.Create(r.Context(), toInput(body))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toResponse(*item))
}

// Update godoc
//
//	@Summary		Update document type
//	@Description	Replaces name, texts and the field-group list. Requires the admin role.
//	@Tags			document-types
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"document type id"
//	@Success		200	{object}	TypeResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Failure		409
//	@Router			/document-types/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body typeBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	item, err := h.types.Update(r.Context(), chi.URLParam(r, "id"), toInput(body))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*item))
}

// Delete godoc
//
//	@Summary		Delete document type
//	@Description	Deletes a document type definition. Requires the admin role.
//	@Tags			document-types
//	@Param			id	path	string	true	"document type id"
//	@Success		204
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Router			/document-types/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	if err := h.types.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeTypeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toInput(body typeBody) documenttype.Input {
	ids := body.FieldGroupIDs
	if ids == nil {
		ids = []string{}
	}
	return documenttype.Input{
		Name:          body.Name,
		Labels:        documenttype.Text{De: body.Labels.De, En: body.Labels.En},
		Description:   documenttype.Text{De: body.Description.De, En: body.Description.En},
		FieldGroupIDs: ids,
	}
}

func toResponse(item documenttype.Type) TypeResponse {
	ids := item.FieldGroupIDs
	if ids == nil {
		ids = []string{}
	}
	return TypeResponse{
		ID:            item.ID,
		Name:          item.Name,
		Labels:        TextResponse{De: item.Labels.De, En: item.Labels.En},
		Description:   TextResponse{De: item.Description.De, En: item.Description.En},
		FieldGroupIDs: ids,
	}
}

func writeTypeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, documenttype.ErrAlreadyExists):
		httputils.Err(w, r, serror.New(http.StatusConflict, "already-exists", "document type already exists"))
	case errors.Is(err, documenttype.ErrNotFound):
		httputils.Err(w, r, serror.NotFound("document-type", "", err))
	case errors.Is(err, documenttype.ErrInvalid):
		httputils.Err(w, r, serror.BadRequest(err, "invalid-document-type", "invalid document type"))
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
