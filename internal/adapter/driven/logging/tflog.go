package logging

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TfLogger adapts terraform-plugin-log/tflog to the port/output.Logger interface.
type TfLogger struct{}

func NewTfLogger() *TfLogger { return &TfLogger{} }

func (l *TfLogger) Debug(ctx context.Context, msg string, fields map[string]any) {
	tflog.Debug(ctx, msg, fields)
}

func (l *TfLogger) Trace(ctx context.Context, msg string, fields map[string]any) {
	tflog.Trace(ctx, msg, fields)
}
