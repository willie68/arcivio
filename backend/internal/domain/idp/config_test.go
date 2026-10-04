package idp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/infrastructure/shttp"
)

func TestCollectRedirectURIsIncludesViteDev(t *testing.T) {
	uris := collectRedirectURIs(config.Config{
		HTTP: shttp.Config{
			ServiceURL: "https://127.0.0.1:9443",
			Sslport:    9443,
		},
	}, nil)
	assert.Contains(t, uris, "https://127.0.0.1:9443/callback")
	assert.Contains(t, uris, "http://localhost:5173/callback")
	assert.Contains(t, uris, "http://127.0.0.1:5173/callback")
}

func TestAllowsLoopbackViteRedirect(t *testing.T) {
	cfg := Config{}
	assert.True(t, cfg.AllowsRedirect("http://localhost:5173/callback"))
	assert.True(t, cfg.AllowsRedirect("http://127.0.0.1:5174/callback"))
	assert.True(t, cfg.AllowsRedirect("http://localhost:5173/callback/"))
	assert.True(t, cfg.AllowsRedirect("http://localhost:5173/client/callback"))
	assert.False(t, cfg.AllowsRedirect("http://evil.example/callback"))
	assert.False(t, cfg.AllowsRedirect("https://localhost:5173/callback"))
	assert.False(t, cfg.AllowsRedirect("http://localhost:5173/other"))
}

func TestOptionalConfigHelpers(t *testing.T) {
	assert.Equal(t, "d", optionalString(nil, "k", "d"))
	assert.Equal(t, "d", optionalString(map[string]any{}, "k", "d"))
	assert.Equal(t, "d", optionalString(map[string]any{"k": nil}, "k", "d"))
	assert.Equal(t, "d", optionalString(map[string]any{"k": 1}, "k", "d"))
	assert.Equal(t, "d", optionalString(map[string]any{"k": ""}, "k", "d"))
	assert.Equal(t, "v", optionalString(map[string]any{"k": "v"}, "k", "d"))

	def := time.Minute
	assert.Equal(t, def, optionalDurationSeconds(nil, "k", def))
	assert.Equal(t, def, optionalDurationSeconds(map[string]any{}, "k", def))
	assert.Equal(t, def, optionalDurationSeconds(map[string]any{"k": nil}, "k", def))
	assert.Equal(t, def, optionalDurationSeconds(map[string]any{"k": 0}, "k", def))
	assert.Equal(t, def, optionalDurationSeconds(map[string]any{"k": "5"}, "k", def))
	assert.Equal(t, 5*time.Second, optionalDurationSeconds(map[string]any{"k": 5}, "k", def))
	assert.Equal(t, 5*time.Second, optionalDurationSeconds(map[string]any{"k": int64(5)}, "k", def))
}
