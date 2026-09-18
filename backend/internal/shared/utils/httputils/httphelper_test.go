package httputils

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/willie68/arcivio/internal/shared/serror"
)

type sampleBody struct {
	Name string `json:"name" validate:"required"`
}

func TestParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := Param(req, "id")
	assert.Error(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	id, err := Param(req, "id")
	assert.NoError(t, err)
	assert.Equal(t, "42", id)
}

func TestDecode(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	var body sampleBody
	assert.NoError(t, Decode(req, &body))
	assert.Equal(t, "ok", body.Name)

	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	assert.Error(t, Decode(req, &sampleBody{}))
}

func TestCreatedAndErr(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	Created(rec, req, "abc", map[string]string{"id": "abc"})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "abc", rec.Header().Get("Location"))

	rec = httptest.NewRecorder()
	Err(rec, req, serror.NotFound("address", "1"))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestFileServer(t *testing.T) {
	r := chi.NewRouter()
	assert.Panics(t, func() {
		FileServer(r, "/{id}", http.Dir("."))
	})

	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("spa-index"), 0o644))
	assert.NoError(t, os.Mkdir(filepath.Join(dir, "assets"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("spa-js"), 0o644))

	FileServer(r, "/client", http.Dir(dir))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/client/", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "spa-index", rec.Body.String())

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/client/assets/app.js", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "spa-js", rec.Body.String())
}
