package domain_test

import (
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestDefaultResolveRequest(t *testing.T) {
	req := domain.DefaultResolveRequest()
	if req.GlobalDir != "_global" {
		t.Errorf("expected GlobalDir=_global, got %q", req.GlobalDir)
	}
	if req.CommonSlug != "common" {
		t.Errorf("expected CommonSlug=common, got %q", req.CommonSlug)
	}
	if req.DefaultsPrefix != "defaults" {
		t.Errorf("expected DefaultsPrefix=defaults, got %q", req.DefaultsPrefix)
	}
	if req.TemplatesKey != "_templates" {
		t.Errorf("expected TemplatesKey=_templates, got %q", req.TemplatesKey)
	}
	if req.InheritKey != "_inherit" {
		t.Errorf("expected InheritKey=_inherit, got %q", req.InheritKey)
	}
	if req.FileExtension != "yaml" {
		t.Errorf("expected FileExtension=yaml, got %q", req.FileExtension)
	}
}

func TestDiscoveredFile(t *testing.T) {
	f := domain.DiscoveredFile{
		Path:     "/config/_global/defaults.common.yaml",
		RelPath:  "_global/defaults.common.yaml",
		Scope:    "_global",
		Prefix:   "defaults",
		Slug:     "common",
		Priority: 1,
	}
	if f.Priority != 1 {
		t.Errorf("expected priority 1, got %d", f.Priority)
	}
}

func TestErrorFormatting(t *testing.T) {
	err := &domain.MergeConflictError{
		Path:        "a.b",
		BaseType:    "map",
		OverlayType: "string",
		BaseFile:    "base.yaml",
		OverlayFile: "overlay.yaml",
	}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}

	err2 := &domain.TemplateNotFoundError{EntryPath: "sqs.orders", TemplateName: "critical"}
	if err2.Error() == "" {
		t.Error("expected non-empty error message")
	}

	err3 := &domain.DuplicateTemplateError{TemplateName: "base"}
	if err3.Error() == "" {
		t.Error("expected non-empty error message")
	}

	err4 := &domain.TemplateWithInheritError{TemplateName: "base", InheritKey: "_inherit"}
	if err4.Error() == "" {
		t.Error("expected non-empty error message")
	}

	err5 := &domain.CaseCollisionError{Dir: "/config", FileA: "App.yaml", FileB: "app.yaml"}
	if err5.Error() == "" {
		t.Error("expected non-empty error message")
	}

	err6 := &domain.MissingVariableError{Key: "DB_PASSWORD", FuncName: "secret"}
	if err6.Error() == "" {
		t.Error("expected non-empty error message")
	}
}
