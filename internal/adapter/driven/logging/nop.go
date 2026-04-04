package logging

import "context"

// NopLogger discards all log output. Use in unit tests.
type NopLogger struct{}

func NewNopLogger() *NopLogger { return &NopLogger{} }

func (l *NopLogger) Debug(_ context.Context, _ string, _ map[string]any) {}
func (l *NopLogger) Trace(_ context.Context, _ string, _ map[string]any) {}
