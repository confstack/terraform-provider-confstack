package domain

import (
	"errors"
	"strings"
)

// ResolveRequest contains all inputs required to resolve a confstack configuration.
type ResolveRequest struct {
	ConfigDir      string
	Environment    string
	Tenant         string
	Variables      map[string]string
	Secrets        map[string]string
	GlobalDir      string
	CommonSlug     string
	DefaultsPrefix string
	TemplatesKey   string
	InheritKey     string
	FileExtension  string
}

// ResolveResult contains the resolved configuration output.
type ResolveResult struct {
	// Output is the fully resolved configuration map (with secrets redacted).
	Output map[string]any
	// SensitiveOutput is the fully resolved configuration map (with secrets in plaintext).
	SensitiveOutput map[string]any
	// LoadedFiles is the ordered list of files that were loaded, in merge priority order.
	LoadedFiles []string
	// SecretPaths records which JSON paths contain secret values.
	SecretPaths map[string]bool
}

// DiscoveredFile represents a single YAML file found during file discovery.
type DiscoveredFile struct {
	// Path is the absolute filesystem path to the file.
	Path string
	// RelPath is the path relative to config_dir.
	RelPath string
	// Scope is either "_global" or the environment name.
	Scope string
	// Prefix is the filename prefix (e.g., "defaults", "networking").
	Prefix string
	// Slug is the tenant/common segment (e.g., "common", "acme").
	Slug string
	// Priority is the 1-8 merge priority level.
	Priority int
}

// NewResolveRequest constructs a validated ResolveRequest with defaults applied.
// configDir and environment are required. Use functional options to override defaults.
func NewResolveRequest(configDir, environment string, opts ...func(*ResolveRequest)) (ResolveRequest, error) {
	if configDir == "" {
		return ResolveRequest{}, errors.New("config_dir is required")
	}
	if environment == "" {
		return ResolveRequest{}, errors.New("environment is required")
	}
	req := ResolveRequest{
		ConfigDir:      configDir,
		Environment:    strings.ToLower(environment),
		GlobalDir:      "_global",
		CommonSlug:     "common",
		DefaultsPrefix: "defaults",
		TemplatesKey:   "_templates",
		InheritKey:     "_inherit",
		FileExtension:  "yaml",
		Variables:      map[string]string{},
		Secrets:        map[string]string{},
	}
	for _, opt := range opts {
		opt(&req)
	}
	req.Tenant = strings.ToLower(req.Tenant)
	return req, nil
}

func WithTenant(t string) func(*ResolveRequest)            { return func(r *ResolveRequest) { r.Tenant = t } }
func WithVariables(v map[string]string) func(*ResolveRequest) { return func(r *ResolveRequest) { r.Variables = v } }
func WithSecrets(s map[string]string) func(*ResolveRequest)   { return func(r *ResolveRequest) { r.Secrets = s } }
func WithGlobalDir(d string) func(*ResolveRequest)         { return func(r *ResolveRequest) { r.GlobalDir = d } }
func WithCommonSlug(s string) func(*ResolveRequest)        { return func(r *ResolveRequest) { r.CommonSlug = s } }
func WithDefaultsPrefix(p string) func(*ResolveRequest)    { return func(r *ResolveRequest) { r.DefaultsPrefix = p } }
func WithTemplatesKey(k string) func(*ResolveRequest)      { return func(r *ResolveRequest) { r.TemplatesKey = k } }
func WithInheritKey(k string) func(*ResolveRequest)        { return func(r *ResolveRequest) { r.InheritKey = k } }
func WithFileExtension(e string) func(*ResolveRequest)     { return func(r *ResolveRequest) { r.FileExtension = e } }

// Deprecated: Use NewResolveRequest instead. DefaultResolveRequest is retained for test convenience.
func DefaultResolveRequest() ResolveRequest {
	return ResolveRequest{
		GlobalDir:      "_global",
		CommonSlug:     "common",
		DefaultsPrefix: "defaults",
		TemplatesKey:   "_templates",
		InheritKey:     "_inherit",
		FileExtension:  "yaml",
		Variables:      map[string]string{},
		Secrets:        map[string]string{},
	}
}
