package output

import "context"

// PathExpander expands a potentially-glob path pattern to a sorted list of concrete file paths.
type PathExpander interface {
	Expand(ctx context.Context, pattern string) ([]string, error)
}
