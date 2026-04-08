package terraform

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMapToTerraformDynamic(t *testing.T) {
	val, err := mapToTerraformDynamic(map[string]any{
		"name":    "app",
		"count":   3,
		"enabled": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.IsNull() {
		t.Fatal("expected non-null dynamic value")
	}
}

func TestMapToTerraformValue_Nil(t *testing.T) {
	val, typ, err := mapToTerraformValue(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val.IsNull() {
		t.Fatal("expected null value")
	}
	if typ != types.DynamicType {
		t.Fatalf("expected dynamic type, got %T", typ)
	}
}

func TestMapToTerraformValue_NestedObjectAndList(t *testing.T) {
	val, _, err := mapToTerraformValue(map[string]any{
		"name": "app",
		"nested": map[string]any{
			"ports": []any{80, "http"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.IsNull() {
		t.Fatal("expected non-null object value")
	}
}

func TestMapToTerraformValue_EmptyList(t *testing.T) {
	val, _, err := mapToTerraformValue([]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.IsNull() {
		t.Fatal("expected non-null tuple value")
	}
}

func TestMapToTerraformValue_UnsupportedFallbackToString(t *testing.T) {
	val, typ, err := mapToTerraformValue(struct{ Name string }{Name: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != types.StringType {
		t.Fatalf("expected string type, got %T", typ)
	}
	if !strings.Contains(val.String(), "x") {
		t.Fatalf("expected stringified fallback value, got %q", val.String())
	}
}

func TestMapToTerraformValue_Primitives(t *testing.T) {
	tests := []any{"value", true, 5, int64(6), 7.5}
	for _, input := range tests {
		val, _, err := mapToTerraformValue(input)
		if err != nil {
			t.Fatalf("unexpected error for %T: %v", input, err)
		}
		if val.IsNull() {
			t.Fatalf("expected non-null value for %T", input)
		}
	}
}

func TestMapToTerraformValue_ObjectTypeRoundTrip(t *testing.T) {
	val, _, err := mapToTerraformValue(map[string]any{
		"service": map[string]any{
			"name": "api",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = val.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("unexpected terraform conversion error: %v", err)
	}
}
