package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// Reader implements port/output.FileReader using the local filesystem.
// It validates that files do not escape configDir via symlinks.
type Reader struct{}

// NewReader returns a new filesystem-backed Reader with symlink protection.
func NewReader() *Reader {
	return &Reader{}
}

// Read reads the file at path after verifying it does not escape configDir via symlinks.
func (r *Reader) Read(ctx context.Context, path string, configDir string) ([]byte, error) {
	// Resolve symlinks for the file path itself
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: path, Cause: err}
	}

	// Resolve symlinks for configDir
	realConfigDir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: configDir, Cause: err}
	}

	// Ensure the resolved path is within configDir
	absConfigDir, err := filepath.Abs(realConfigDir)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: configDir, Cause: err}
	}
	absRealPath, err := filepath.Abs(realPath)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: realPath, Cause: err}
	}

	if !strings.HasPrefix(absRealPath, absConfigDir+string(os.PathSeparator)) &&
		absRealPath != absConfigDir {
		return nil, &domain.SymlinkEscapeError{
			FilePath:   path,
			ResolvedTo: absRealPath,
			ConfigDir:  absConfigDir,
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &domain.FileReadError{FilePath: path, Cause: err}
	}
	return data, nil
}
