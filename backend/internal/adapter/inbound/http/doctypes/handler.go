package doctypes

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	"github.com/willie68/arcivio/internal/domain/doctype"
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

// TypeResponse is one document type.
type TypeResponse struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
	FieldGroups []string     `json:"fieldGroups"`
}

// TypeListResponse is the document-type catalog.
type TypeListResponse struct {
	Items []TypeResponse `json:"items"`
}

type typeBody struct {
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
	FieldGroups []string     `json:"fieldGroups"`
}

type typeService interface {
	List(ctx context.Context) ([]doctype.Type, error)
	Get(ctx context.Context, id string) (*doctype.Type, error)
	Create(ctx context.Context, in doctype.Input) (*doctype.Type, error)
	Update(ctx context.Context, id string, in doctype.Input) (*doctype.Type, error)
	Delete(ctx context.Context, id string) error
	Export(ctx context.Context) (doctype.Exchange, error)
	Preview(ctx context.Context, ex doctype.Exchange) ([]doctype.Conflict, error)
	Import(ctx context.Context, ex doctype.Exchange, decisions []doctype.Decision) (doctype.ImportResult, error)
}

type rolesService interface {
	HasRole(checkRole string, desiredRole string) bool
}

// Handler serves document types under /document-types.
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
	r.Get("/export", h.Export)
	r.Post("/import/preview", h.PreviewImport)
	r.Post("/import", h.Import)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return "/document-types", r
}

// List godoc
//
//	@Summary		List document types
//	@Description	Returns document types. Requires the admin role.
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
	for _, docType := range listed {
		items = append(items, toResponse(docType))
	}
	render.JSON(w, r, TypeListResponse{Items: items})
}

// Get godoc
//
//	@Summary		Get document type
//	@Description	Returns one document type. Requires the admin role.
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
	docType, err := h.types.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*docType))
}

// Create godoc
//
//	@Summary		Create document type
//	@Description	Stores a document type. The system field group must be included. Requires the admin role.
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
	docType, err := h.types.Create(r.Context(), toInput(body))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toResponse(*docType))
}

// Update godoc
//
//	@Summary		Update document type
//	@Description	Replaces name, texts and field groups. The system field group cannot be removed. Requires the admin role.
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
	docType, err := h.types.Update(r.Context(), chi.URLParam(r, "id"), toInput(body))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, toResponse(*docType))
}

// Delete godoc
//
//	@Summary		Delete document type
//	@Description	Deletes a document type. Requires the admin role.
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

// Export godoc
//
//	@Summary		Export document types
//	@Description	Returns document types and their field groups as JSON. Readonly system groups are omitted. Requires the admin role.
//	@Tags			document-types
//	@Produce		json
//	@Success		200	{object}	exchangeResponse
//	@Failure		401
//	@Failure		403
//	@Router			/document-types/export [get]
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	ex, err := h.types.Export(r.Context())
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, toExchangeResponse(ex))
}

// Import godoc
//
//	@Summary		Import document types
//	@Description	Creates or updates document types and field groups from JSON. Readonly system groups are left unchanged. Requires the admin role.
//	@Tags			document-types
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	importResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Router			/document-types/import [post]
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body importBody
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	result, err := h.types.Import(r.Context(), fromExchangeResponse(body.exchangeResponse), fromDecisions(body.Decisions))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, importResponse{
		DocumentTypes: result.DocumentTypes,
		FieldGroups:   result.FieldGroups,
	})
}

// PreviewImport godoc
//
//	@Summary		Preview a document type import
//	@Description	Reports id and name conflicts. Nothing is written. Requires the admin role.
//	@Tags			document-types
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	previewResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Router			/document-types/import/preview [post]
func (h *Handler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.roles, w, r) {
		return
	}
	var body exchangeResponse
	if err := httputils.Decode(r, &body); err != nil {
		httputils.Err(w, r, err)
		return
	}
	conflicts, err := h.types.Preview(r.Context(), fromExchangeResponse(body))
	if err != nil {
		writeTypeError(w, r, err)
		return
	}
	render.JSON(w, r, previewResponse{Conflicts: toConflictResponses(conflicts)})
}

type exchangeResponse struct {
	DocumentTypes []exchangeTypeBody  `json:"documentTypes"`
	FieldGroups   []exchangeGroupBody `json:"fieldGroups"`
}

type exchangeTypeBody struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
	FieldGroups []string     `json:"fieldGroups"`
}

type exchangeGroupBody struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Labels      TextResponse        `json:"labels"`
	Description TextResponse        `json:"description"`
	Fields      []exchangeFieldBody `json:"fields"`
}

type exchangeFieldBody struct {
	Name        string       `json:"name"`
	Labels      TextResponse `json:"labels"`
	Description TextResponse `json:"description"`
	ValueType   string       `json:"valueType"`
	Mandatory   bool         `json:"mandatory"`
}

type importBody struct {
	exchangeResponse
	Decisions []decisionBody `json:"decisions"`
}

type decisionBody struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Action string `json:"action"`
	Name   string `json:"name"`
}

type importResponse struct {
	DocumentTypes int `json:"documentTypes"`
	FieldGroups   int `json:"fieldGroups"`
}

type previewResponse struct {
	Conflicts []conflictResponse `json:"conflicts"`
}

type conflictResponse struct {
	Kind          string           `json:"kind"`
	Reason        string           `json:"reason"`
	ID            string           `json:"id"`
	Exists        bool             `json:"exists"`
	Name          string           `json:"name"`
	ExistingID    string           `json:"existingId"`
	ExistingName  string           `json:"existingName"`
	SuggestedName string           `json:"suggestedName"`
	Changes       []changeResponse `json:"changes"`
}

type changeResponse struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

func toExchangeResponse(ex doctype.Exchange) exchangeResponse {
	types := make([]exchangeTypeBody, 0, len(ex.DocumentTypes))
	for _, docType := range ex.DocumentTypes {
		groups := docType.FieldGroups
		if groups == nil {
			groups = []string{}
		}
		types = append(types, exchangeTypeBody{
			ID:          docType.ID,
			Name:        docType.Name,
			Labels:      TextResponse{De: docType.Labels.De, En: docType.Labels.En},
			Description: TextResponse{De: docType.Description.De, En: docType.Description.En},
			FieldGroups: groups,
		})
	}
	groups := make([]exchangeGroupBody, 0, len(ex.FieldGroups))
	for _, group := range ex.FieldGroups {
		fields := make([]exchangeFieldBody, 0, len(group.Fields))
		for _, field := range group.Fields {
			fields = append(fields, exchangeFieldBody{
				Name:        field.Name,
				Labels:      TextResponse{De: field.Labels.De, En: field.Labels.En},
				Description: TextResponse{De: field.Description.De, En: field.Description.En},
				ValueType:   field.ValueType,
				Mandatory:   field.Mandatory,
			})
		}
		groups = append(groups, exchangeGroupBody{
			ID:          group.ID,
			Name:        group.Name,
			Labels:      TextResponse{De: group.Labels.De, En: group.Labels.En},
			Description: TextResponse{De: group.Description.De, En: group.Description.En},
			Fields:      fields,
		})
	}
	return exchangeResponse{DocumentTypes: types, FieldGroups: groups}
}

func fromExchangeResponse(body exchangeResponse) doctype.Exchange {
	types := make([]doctype.ExchangeType, 0, len(body.DocumentTypes))
	for _, docType := range body.DocumentTypes {
		types = append(types, doctype.ExchangeType{
			ID:          docType.ID,
			Name:        docType.Name,
			Labels:      doctype.Text{De: docType.Labels.De, En: docType.Labels.En},
			Description: doctype.Text{De: docType.Description.De, En: docType.Description.En},
			FieldGroups: docType.FieldGroups,
		})
	}
	groups := make([]doctype.ExchangeGroup, 0, len(body.FieldGroups))
	for _, group := range body.FieldGroups {
		fields := make([]doctype.ExchangeField, 0, len(group.Fields))
		for _, field := range group.Fields {
			fields = append(fields, doctype.ExchangeField{
				Name:        field.Name,
				Labels:      doctype.Text{De: field.Labels.De, En: field.Labels.En},
				Description: doctype.Text{De: field.Description.De, En: field.Description.En},
				ValueType:   field.ValueType,
				Mandatory:   field.Mandatory,
			})
		}
		groups = append(groups, doctype.ExchangeGroup{
			ID:          group.ID,
			Name:        group.Name,
			Labels:      doctype.Text{De: group.Labels.De, En: group.Labels.En},
			Description: doctype.Text{De: group.Description.De, En: group.Description.En},
			Fields:      fields,
		})
	}
	return doctype.Exchange{DocumentTypes: types, FieldGroups: groups}
}

func fromDecisions(raw []decisionBody) []doctype.Decision {
	decisions := make([]doctype.Decision, 0, len(raw))
	for _, decision := range raw {
		decisions = append(decisions, doctype.Decision{
			Kind:   decision.Kind,
			ID:     decision.ID,
			Action: decision.Action,
			Name:   decision.Name,
		})
	}
	return decisions
}

func toConflictResponses(conflicts []doctype.Conflict) []conflictResponse {
	out := make([]conflictResponse, 0, len(conflicts))
	for _, conflict := range conflicts {
		changes := make([]changeResponse, 0, len(conflict.Changes))
		for _, change := range conflict.Changes {
			changes = append(changes, changeResponse{
				Field:  change.Field,
				Before: change.Before,
				After:  change.After,
			})
		}
		out = append(out, conflictResponse{
			Kind:          conflict.Kind,
			Reason:        conflict.Reason,
			ID:            conflict.ID,
			Exists:        conflict.Exists,
			Name:          conflict.Name,
			ExistingID:    conflict.ExistingID,
			ExistingName:  conflict.ExistingName,
			SuggestedName: conflict.SuggestedName,
			Changes:       changes,
		})
	}
	return out
}

func toInput(body typeBody) doctype.Input {
	return doctype.Input{
		Name:        body.Name,
		Labels:      doctype.Text{De: body.Labels.De, En: body.Labels.En},
		Description: doctype.Text{De: body.Description.De, En: body.Description.En},
		FieldGroups: body.FieldGroups,
	}
}

func toResponse(docType doctype.Type) TypeResponse {
	groups := docType.FieldGroups
	if groups == nil {
		groups = []string{}
	}
	return TypeResponse{
		ID:          docType.ID,
		Name:        docType.Name,
		Labels:      TextResponse{De: docType.Labels.De, En: docType.Labels.En},
		Description: TextResponse{De: docType.Description.De, En: docType.Description.En},
		FieldGroups: groups,
	}
}

func writeTypeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, doctype.ErrAlreadyExists):
		httputils.Err(w, r, serror.New(http.StatusConflict, "already-exists", "document type already exists"))
	case errors.Is(err, doctype.ErrNotFound):
		httputils.Err(w, r, serror.NotFound("document-type", "", err))
	case errors.Is(err, doctype.ErrSystemGroup):
		httputils.Err(w, r, serror.BadRequest(err, "system-group", "the system field group cannot be removed"))
	case errors.Is(err, doctype.ErrUnknownGroup):
		httputils.Err(w, r, serror.BadRequest(err, "unknown-field-group", "unknown field group"))
	case errors.Is(err, doctype.ErrInvalid):
		httputils.Err(w, r, serror.BadRequest(err, "invalid-document-type", "invalid document type"))
	case errors.Is(err, doctype.ErrConflict):
		httputils.Err(w, r, serror.New(http.StatusConflict, "import-conflict", "import needs a decision"))
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
