package output

import "context"

// Logger provides structured logging for the resolution pipeline.
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
	Trace(ctx context.Context, msg string, fields map[string]any)
}
