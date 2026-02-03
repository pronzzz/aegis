package storage

import (
	"os"
	"path/filepath"
)

// LocalBackend implements Backend for local filesystem
type LocalBackend struct {
	BasePath string
}

func NewLocalBackend(basePath string) (*LocalBackend, error) {
	// Ensure objects dir exists
	if err := os.MkdirAll(filepath.Join(basePath, "objects"), 0700); err != nil {
		return nil, err
	}
	return &LocalBackend{BasePath: basePath}, nil
}

func (l *LocalBackend) objectPath(key string) string {
	if len(key) < 2 {
		return filepath.Join(l.BasePath, "objects", key)
	}
	return filepath.Join(l.BasePath, "objects", key[:2], key[2:])
}

func (l *LocalBackend) Put(key string, data []byte) error {
	path := l.objectPath(key)

	// Check exist
	if _, err := os.Stat(path); err == nil {
		return nil // Already exists
	}

	// Create dir (e.g. objects/ab/)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (l *LocalBackend) Get(key string) ([]byte, error) {
	path := l.objectPath(key)
	return os.ReadFile(path)
}

func (l *LocalBackend) Has(key string) (bool, error) {
	path := l.objectPath(key)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l *LocalBackend) Close() error {
	return nil
}
