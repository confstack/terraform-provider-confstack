package input

import (
	"context"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// ConfigResolver is the primary input port. It orchestrates the full configuration resolution pipeline.
type ConfigResolver interface {
	Resolve(ctx context.Context, req domain.ResolveRequest) (*domain.ResolveResult, error)
}
