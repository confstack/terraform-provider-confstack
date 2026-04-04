package output

import "context"

// FileReader reads the raw bytes of a file, validating that it does not escape configDir via symlinks.
type FileReader interface {
	Read(ctx context.Context, path string, configDir string) ([]byte, error)
}
