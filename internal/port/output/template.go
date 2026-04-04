package output

import (
	"context"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// TemplateEngine processes a file as a Go template, injecting variables and tracking secrets via sentinels.
// It returns:
//   - processed: the rendered template bytes (with sentinel strings substituted for secrets)
//   - sentinelMap: maps sentinel string → real secret value
type TemplateEngine interface {
	Process(ctx context.Context, data []byte, filePath string, req domain.ResolveRequest, nonce string) (processed []byte, sentinelMap map[string]string, err error)
}
