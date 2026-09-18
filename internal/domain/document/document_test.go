package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type pingStore struct{ err error }

func (p pingStore) Ping() error { return p.err }

func TestDocumentsStatus(t *testing.T) {
	d := NewDocuments(pingStore{})
	st, err := d.Status()
	assert.NoError(t, err)
	assert.Equal(t, "ok", st)
}
