package output

import "context"

// FileReader reads the raw bytes of a file.
type FileReader interface {
	Read(ctx context.Context, path string) ([]byte, error)
}
