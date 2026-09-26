package storage

import (
	"os"
	"path/filepath"
)

// Local is a Backend that stores files under a root directory on the
// local filesystem (typically ~/.anchor/data/).
type Local struct {
	root string
}

// NewLocal returns a Local backend rooted at root, creating it if needed.
func NewLocal(root string) (*Local, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}

func (l *Local) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(path)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (l *Local) WriteFile(path string, data []byte) error {
	full := filepath.Join(l.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}
