// Package filesystem defines a storage-driver-agnostic contract so modules
// can read/write files without depending on local disk, S3, etc. directly.
package filesystem

import (
	"io"
	"os"
	"path/filepath"
)

type Filesystem interface {
	Put(path string, r io.Reader) error
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) bool
}

// Local is a Filesystem implementation backed by the local disk, rooted
// at basePath (e.g. the storage/ directory).
type Local struct {
	basePath string
}

func NewLocal(basePath string) *Local {
	return &Local{basePath: basePath}
}

func (l *Local) resolve(path string) string {
	return filepath.Join(l.basePath, path)
}

func (l *Local) Put(path string, r io.Reader) error {
	full := l.resolve(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Get(path string) (io.ReadCloser, error) {
	return os.Open(l.resolve(path))
}

func (l *Local) Delete(path string) error {
	return os.Remove(l.resolve(path))
}

func (l *Local) Exists(path string) bool {
	_, err := os.Stat(l.resolve(path))
	return err == nil
}
