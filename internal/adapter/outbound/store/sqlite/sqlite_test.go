package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPingClose(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "a.db"))
	assert.NoError(t, err)
	assert.NoError(t, st.Ping())
	assert.NoError(t, st.Shutdown())
}
