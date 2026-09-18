package document

// Store is the outbound port for the embedded metadata database.
type Store interface {
	Ping() error
}

// Documents is the document use-case root (Phase 1: health of the store only).
type Documents struct {
	store Store
}

// NewDocuments creates the document use cases.
func NewDocuments(store Store) *Documents {
	return &Documents{store: store}
}

// Status reports whether the metadata store is reachable.
func (d *Documents) Status() (string, error) {
	if err := d.store.Ping(); err != nil {
		return "", err
	}
	return "ok", nil
}
