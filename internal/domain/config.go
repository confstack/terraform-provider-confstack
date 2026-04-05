package domain

import (
	"errors"
	"fmt"
)

// ResolveRequest contains all inputs required to resolve a confstack configuration.
type ResolveRequest struct {
	// Layers is the ordered list of YAML file paths to load and merge.
	// Index 0 is lowest priority; the last entry is highest priority (last wins).
	Layers []string
	// OnMissingLayer controls behavior when a layer file does not exist.
	// Valid values: "error" (default), "warn", "skip".
	OnMissingLayer string
	// Variables are available to {{ var "KEY" }} template functions.
	Variables map[string]string
	// Secrets are available to {{ secret "KEY" }} template functions.
	Secrets map[string]string
	// TemplatesKey is the reserved YAML key for template definitions. Default: "_templates".
	TemplatesKey string
	// InheritKey is the reserved YAML key for inheritance directives. Default: "_inherit".
	InheritKey string
	// FlatSeparator is used when flattening nested keys into flat_config. Default: ".".
	FlatSeparator string
}

// ResolveResult contains the resolved configuration output.
type ResolveResult struct {
	// Output is the fully resolved configuration map (with secrets redacted).
	Output map[string]any
	// SensitiveOutput is the fully resolved configuration map (with secrets in plaintext).
	SensitiveOutput map[string]any
	// FlatOutput is Output flattened to separator-delimited string keys.
	FlatOutput map[string]any
	// LoadedLayers is the ordered list of layer paths that were successfully loaded.
	LoadedLayers []string
	// SecretPaths records which dot-delimited paths contain secret values.
	SecretPaths map[string]bool
}

// DiscoveredFile represents a single YAML layer file to be processed.
type DiscoveredFile struct {
	// Path is the filesystem path to the file.
	Path string
	// Priority is the merge priority (index in the layers list; 0 = lowest).
	Priority int
}

// NewResolveRequest constructs a validated ResolveRequest with defaults applied.
// At least one layer is required.
func NewResolveRequest(layers []string, opts ...func(*ResolveRequest)) (ResolveRequest, error) {
	if len(layers) == 0 {
		return ResolveRequest{}, errors.New("layers must not be empty")
	}
	req := ResolveRequest{
		Layers:        layers,
		OnMissingLayer:  "error",
		TemplatesKey:  "_templates",
		InheritKey:    "_inherit",
		FlatSeparator: ".",
		Variables:     map[string]string{},
		Secrets:       map[string]string{},
	}
	for _, opt := range opts {
		opt(&req)
	}
	switch req.OnMissingLayer {
	case "error", "warn", "skip":
		// valid
	default:
		return ResolveRequest{}, fmt.Errorf("on_missing_layer must be one of \"error\", \"warn\", \"skip\"; got %q", req.OnMissingLayer)
	}
	return req, nil
}

func WithOnMissingLayer(m string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.OnMissingLayer = m }
}
func WithVariables(v map[string]string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.Variables = v }
}
func WithSecrets(s map[string]string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.Secrets = s }
}
func WithTemplatesKey(k string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.TemplatesKey = k }
}
func WithInheritKey(k string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.InheritKey = k }
}
func WithFlatSeparator(s string) func(*ResolveRequest) {
	return func(r *ResolveRequest) { r.FlatSeparator = s }
}
