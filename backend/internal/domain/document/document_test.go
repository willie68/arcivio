package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentsStatus(t *testing.T) {
	store := newMockStore(t)
	store.EXPECT().Ping().Return(nil)
	d := NewDocuments(store)
	st, err := d.Status()
	assert.NoError(t, err)
	assert.Equal(t, "ok", st)
}
