package domain_test

import (
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestNewResolveRequest_Defaults(t *testing.T) {
	req, err := domain.NewResolveRequest([]string{"a.yaml", "b.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.OnMissingLayer != "error" {
		t.Errorf("expected OnMissingLayer=error, got %q", req.OnMissingLayer)
	}
	if req.TemplatesKey != "_templates" {
		t.Errorf("expected TemplatesKey=_templates, got %q", req.TemplatesKey)
	}
	if req.InheritKey != "_inherit" {
		t.Errorf("expected InheritKey=_inherit, got %q", req.InheritKey)
	}
	if req.FlatSeparator != "." {
		t.Errorf("expected FlatSeparator=., got %q", req.FlatSeparator)
	}
	if len(req.Layers) != 2 {
		t.Errorf("expected 2 layers, got %d", len(req.Layers))
	}
}

func TestNewResolveRequest_EmptyLayers(t *testing.T) {
	_, err := domain.NewResolveRequest([]string{})
	if err == nil {
		t.Error("expected error for empty layers")
	}
}

func TestNewResolveRequest_InvalidOnMissingLayer(t *testing.T) {
	_, err := domain.NewResolveRequest(
		[]string{"a.yaml"},
		domain.WithOnMissingLayer("bogus"),
	)
	if err == nil {
		t.Error("expected error for invalid on_missing_layer")
	}
}

func TestNewResolveRequest_ValidOnMissingLayerValues(t *testing.T) {
	for _, v := range []string{"error", "warn", "skip"} {
		_, err := domain.NewResolveRequest(
			[]string{"a.yaml"},
			domain.WithOnMissingLayer(v),
		)
		if err != nil {
			t.Errorf("unexpected error for on_missing_layer=%q: %v", v, err)
		}
	}
}

func TestNewResolveRequest_WithOptions(t *testing.T) {
	req, err := domain.NewResolveRequest(
		[]string{"base.yaml"},
		domain.WithOnMissingLayer("skip"),
		domain.WithVariables(map[string]string{"FOO": "bar"}),
		domain.WithFlatSeparator("/"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.OnMissingLayer != "skip" {
		t.Errorf("expected OnMissingLayer=skip, got %q", req.OnMissingLayer)
	}
	if req.Variables["FOO"] != "bar" {
		t.Errorf("expected Variables[FOO]=bar, got %q", req.Variables["FOO"])
	}
	if req.FlatSeparator != "/" {
		t.Errorf("expected FlatSeparator=/, got %q", req.FlatSeparator)
	}
}

func TestIsGlobPattern(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"foo.yaml", false},
		{"path/to/file.yaml", false},
		{"*.yaml", true},
		{"**/*.yaml", true},
		{"file[0-9].yaml", true},
		{"file?.yaml", true},
		{"", false},
	}
	for _, tt := range tests {
		got := domain.IsGlobPattern(tt.input)
		if got != tt.want {
			t.Errorf("IsGlobPattern(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDiscoveredFile(t *testing.T) {
	f := domain.DiscoveredFile{
		Path:     "/config/base.yaml",
		Priority: 0,
	}
	if f.Priority != 0 {
		t.Errorf("expected priority 0, got %d", f.Priority)
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
	if err.Error() == "" {
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

	err5 := &domain.LayerNotFoundError{LayerPath: "/config/missing.yaml"}
	if err5.Error() == "" {
		t.Error("expected non-empty error message")
	}

	err6 := &domain.MissingVariableError{Key: "DB_PASSWORD", FuncName: "secret"}
	if err6.Error() == "" {
		t.Error("expected non-empty error message")
	}
}
