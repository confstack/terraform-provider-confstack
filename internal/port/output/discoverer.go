package output

import (
	"context"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// FileDiscoverer scans a config directory and returns the ordered list of files to load.
type FileDiscoverer interface {
	Discover(ctx context.Context, req domain.ResolveRequest) ([]domain.DiscoveredFile, error)
}
