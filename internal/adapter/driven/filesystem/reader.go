package filesystem

import (
	"context"
	"os"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// Reader implements port/output.FileReader using the local filesystem.
type Reader struct{}

// NewReader returns a new filesystem-backed Reader.
func NewReader() *Reader {
	return &Reader{}
}

// Read reads the file at path and returns its contents.
func (r *Reader) Read(_ context.Context, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: path, Cause: err}
	}
	return data, nil
}
