package store

// Config selects the embedded store implementation.
type Config struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}
