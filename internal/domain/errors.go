package domain

import "fmt"

// MergeConflictError is returned when a deep merge encounters incompatible types at the same path.
type MergeConflictError struct {
	Path        string
	BaseType    string
	OverlayType string
	BaseFile    string
	OverlayFile string
}

func (e *MergeConflictError) Error() string {
	return fmt.Sprintf("merge conflict at path %q: cannot merge %s (from %s) with %s (from %s)",
		e.Path, e.BaseType, e.BaseFile, e.OverlayType, e.OverlayFile)
}

// TemplateNotFoundError is returned when an _inherit directive references a non-existent template.
type TemplateNotFoundError struct {
	EntryPath    string
	TemplateName string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("entry %q references template %q which does not exist in %q",
		e.EntryPath, e.TemplateName, "_templates")
}

// DuplicateTemplateError is returned when the same template name is defined in multiple _templates blocks.
type DuplicateTemplateError struct {
	TemplateName string
}

func (e *DuplicateTemplateError) Error() string {
	return fmt.Sprintf("template name %q is defined multiple times; template names must be globally unique", e.TemplateName)
}

// TemplateWithInheritError is returned when a template definition contains an _inherit key.
type TemplateWithInheritError struct {
	TemplateName string
	InheritKey   string
}

func (e *TemplateWithInheritError) Error() string {
	return fmt.Sprintf("template %q cannot contain %q; templates must not inherit from other templates",
		e.TemplateName, e.InheritKey)
}

// MissingVariableError is returned when a var() or secret() template function references a key not found in inputs or environment.
type MissingVariableError struct {
	Key      string
	FuncName string
}

func (e *MissingVariableError) Error() string {
	return fmt.Sprintf("template function %s(%q): key not found in provided map or environment variables",
		e.FuncName, e.Key)
}

// ParseError is returned when a configuration file cannot be parsed.
type ParseError struct {
	FilePath string
	Detail   string
	Cause    error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("parse error in %q: %s: %v", e.FilePath, e.Detail, e.Cause)
	}
	return fmt.Sprintf("parse error in %q: %s", e.FilePath, e.Detail)
}

func (e *ParseError) Unwrap() error { return e.Cause }

// FileReadError is returned when a file cannot be read.
type FileReadError struct {
	FilePath string
	Cause    error
}

func (e *FileReadError) Error() string {
	return fmt.Sprintf("reading %q: %v", e.FilePath, e.Cause)
}

func (e *FileReadError) Unwrap() error { return e.Cause }

// LayerNotFoundError is returned when a layer file does not exist and on_missing_layer = "error".
type LayerNotFoundError struct {
	LayerPath string
}

func (e *LayerNotFoundError) Error() string {
	return fmt.Sprintf("layer file %q does not exist", e.LayerPath)
}

// NoGlobMatchError is returned when a glob pattern matches zero files and on_missing_layer = "error".
type NoGlobMatchError struct {
	Pattern string
}

func (e *NoGlobMatchError) Error() string {
	return fmt.Sprintf("glob pattern %q matched no files", e.Pattern)
}

// TemplateRenderError is returned when template processing fails.
type TemplateRenderError struct {
	FilePath string
	Detail   string
	Cause    error
}

func (e *TemplateRenderError) Error() string {
	return fmt.Sprintf("template %s error in %q: %v", e.Detail, e.FilePath, e.Cause)
}

func (e *TemplateRenderError) Unwrap() error { return e.Cause }
