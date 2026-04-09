package filesystem

import (
	"context"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
)

// Expander implements port/output.PathExpander using doublestar glob matching.
type Expander struct{}

// NewExpander returns a new filesystem-backed Expander.
func NewExpander() *Expander {
	return &Expander{}
}

// Expand expands pattern to a sorted list of matching file paths.
// Returns an empty slice if no files match.
func (e *Expander) Expand(_ context.Context, pattern string) ([]string, error) {
	matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly())
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
