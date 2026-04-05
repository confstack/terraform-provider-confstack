package domain_test

import (
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestFlatten_Simple(t *testing.T) {
	data := map[string]any{
		"a": "hello",
		"b": 42,
	}
	got := domain.Flatten(data, ".")
	if got["a"] != "hello" {
		t.Errorf("expected a=hello, got %v", got["a"])
	}
	if got["b"] != 42 {
		t.Errorf("expected b=42, got %v", got["b"])
	}
}

func TestFlatten_Nested(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
			"credentials": map[string]any{
				"user": "admin",
			},
		},
		"app": "myapp",
	}
	got := domain.Flatten(data, ".")

	if got["database.host"] != "localhost" {
		t.Errorf("expected database.host=localhost, got %v", got["database.host"])
	}
	if got["database.port"] != 5432 {
		t.Errorf("expected database.port=5432, got %v", got["database.port"])
	}
	if got["database.credentials.user"] != "admin" {
		t.Errorf("expected database.credentials.user=admin, got %v", got["database.credentials.user"])
	}
	if got["app"] != "myapp" {
		t.Errorf("expected app=myapp, got %v", got["app"])
	}
	// No intermediate keys
	if _, ok := got["database"]; ok {
		t.Error("expected 'database' intermediate key to be absent from flat output")
	}
}

func TestFlatten_CustomSeparator(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{"b": "val"},
	}
	got := domain.Flatten(data, "/")
	if got["a/b"] != "val" {
		t.Errorf("expected a/b=val, got %v", got["a/b"])
	}
}

func TestFlatten_ListLeaf(t *testing.T) {
	list := []any{"x", "y"}
	data := map[string]any{
		"items": list,
	}
	got := domain.Flatten(data, ".")
	if _, ok := got["items"]; !ok {
		t.Error("expected items to be present as leaf (list not recursed)")
	}
}

func TestFlatten_Empty(t *testing.T) {
	got := domain.Flatten(map[string]any{}, ".")
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}
