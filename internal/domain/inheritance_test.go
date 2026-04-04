package domain_test

import (
	"errors"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

const templatesKey = "_templates"
const inheritKey = "_inherit"

func TestInheritance_SimpleString(t *testing.T) {
	tree := map[string]any{
		"sqs_queues": map[string]any{
			"_templates": map[string]any{
				"standard": map[string]any{
					"retention":          86400,
					"dlq":                true,
					"visibility_timeout": 30,
				},
			},
			"notifications": map[string]any{
				"_inherit":  "standard",
				"retention": 3600,
			},
		},
	}

	result, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err != nil {
		t.Fatal(err)
	}

	sq := result["sqs_queues"].(map[string]any)
	notif := sq["notifications"].(map[string]any)

	if notif["retention"] != 3600 {
		t.Errorf("expected retention=3600 (entry override), got %v", notif["retention"])
	}
	if notif["dlq"] != true {
		t.Errorf("expected dlq=true from template, got %v", notif["dlq"])
	}
	if notif["visibility_timeout"] != 30 {
		t.Errorf("expected visibility_timeout=30 from template, got %v", notif["visibility_timeout"])
	}
}

func TestInheritance_MultipleWithExcept(t *testing.T) {
	tree := map[string]any{
		"sqs_queues": map[string]any{
			"_templates": map[string]any{
				"standard": map[string]any{
					"retention":          86400,
					"dlq":                true,
					"visibility_timeout": 30,
				},
				"critical": map[string]any{
					"retention":          604800,
					"dlq":                true,
					"visibility_timeout": 30,
					"dlq_max_retries":    5,
				},
			},
			"orders": map[string]any{
				"_inherit": []any{
					map[string]any{
						"template": "standard",
						"except":   []any{"dlq"},
					},
					map[string]any{
						"template": "critical",
					},
				},
				"visibility_timeout": 120,
			},
		},
	}

	result, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err != nil {
		t.Fatal(err)
	}

	sq := result["sqs_queues"].(map[string]any)
	orders := sq["orders"].(map[string]any)

	if orders["retention"] != 604800 {
		t.Errorf("expected retention=604800 from critical, got %v", orders["retention"])
	}
	if orders["visibility_timeout"] != 120 {
		t.Errorf("expected visibility_timeout=120 from entry override, got %v", orders["visibility_timeout"])
	}
	if orders["dlq"] != true {
		t.Errorf("expected dlq=true from critical (overrides except from standard), got %v", orders["dlq"])
	}
	if orders["dlq_max_retries"] != 5 {
		t.Errorf("expected dlq_max_retries=5 from critical, got %v", orders["dlq_max_retries"])
	}
}

func TestInheritance_DuplicateTemplate(t *testing.T) {
	tree := map[string]any{
		"a": map[string]any{
			"_templates": map[string]any{
				"base": map[string]any{"x": 1},
			},
		},
		"b": map[string]any{
			"_templates": map[string]any{
				"base": map[string]any{"y": 2},
			},
		},
	}

	_, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err == nil {
		t.Fatal("expected error for duplicate template name")
	}
	var dte *domain.DuplicateTemplateError
	if !errors.As(err, &dte) {
		t.Errorf("expected DuplicateTemplateError, got %T: %v", err, err)
	}
}

func TestInheritance_TemplateWithInherit(t *testing.T) {
	tree := map[string]any{
		"_templates": map[string]any{
			"bad": map[string]any{
				"_inherit": "other",
				"x":        1,
			},
		},
	}

	_, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err == nil {
		t.Fatal("expected error for template containing _inherit")
	}
	var twie *domain.TemplateWithInheritError
	if !errors.As(err, &twie) {
		t.Errorf("expected TemplateWithInheritError, got %T: %v", err, err)
	}
}

func TestInheritance_MissingTemplate(t *testing.T) {
	tree := map[string]any{
		"queues": map[string]any{
			"_templates": map[string]any{
				"existing": map[string]any{"x": 1},
			},
			"entry": map[string]any{
				"_inherit": "nonexistent",
			},
		},
	}

	_, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err == nil {
		t.Fatal("expected error for missing template reference")
	}
	var tnfe *domain.TemplateNotFoundError
	if !errors.As(err, &tnfe) {
		t.Errorf("expected TemplateNotFoundError, got %T: %v", err, err)
	}
}

func TestInheritance_BubbleUp(t *testing.T) {
	// Template defined at parent level, _inherit used in child
	tree := map[string]any{
		"_templates": map[string]any{
			"global_base": map[string]any{
				"region": "us-east-1",
			},
		},
		"resources": map[string]any{
			"bucket": map[string]any{
				"_inherit": "global_base",
				"name":     "my-bucket",
			},
		},
	}

	result, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err != nil {
		t.Fatal(err)
	}

	bucket := result["resources"].(map[string]any)["bucket"].(map[string]any)
	if bucket["region"] != "us-east-1" {
		t.Errorf("expected region=us-east-1 from bubble-up, got %v", bucket["region"])
	}
	if bucket["name"] != "my-bucket" {
		t.Errorf("expected name=my-bucket, got %v", bucket["name"])
	}
}

func TestInheritance_NestedDepth(t *testing.T) {
	tree := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"_templates": map[string]any{
						"deep_tmpl": map[string]any{"deep_key": "deep_val"},
					},
					"entry": map[string]any{
						"_inherit": "deep_tmpl",
						"own_key":  "own_val",
					},
				},
			},
		},
	}

	result, err := domain.ResolveInheritance(tree, templatesKey, inheritKey)
	if err != nil {
		t.Fatal(err)
	}

	entry := result["level1"].(map[string]any)["level2"].(map[string]any)["level3"].(map[string]any)["entry"].(map[string]any)
	if entry["deep_key"] != "deep_val" {
		t.Errorf("expected deep_key=deep_val, got %v", entry["deep_key"])
	}
	if entry["own_key"] != "own_val" {
		t.Errorf("expected own_key=own_val, got %v", entry["own_key"])
	}
}

func TestInheritance_ListOfStrings(t *testing.T) {
	tree := map[string]any{
		"_templates": map[string]any{
			"tmpl_a": map[string]any{"a": 1},
			"tmpl_b": map[string]any{"b": 2},
		},
		"entry": map[string]any{
			"_inherit": []any{"tmpl_a", "tmpl_b"},
			"c":        3,
		},
	}

	result, err := domain.ResolveInheritance(tree, "_templates", "_inherit")
	if err != nil {
		t.Fatal(err)
	}

	entry := result["entry"].(map[string]any)
	if entry["a"] != 1 {
		t.Errorf("expected a=1 from tmpl_a, got %v", entry["a"])
	}
	if entry["b"] != 2 {
		t.Errorf("expected b=2 from tmpl_b, got %v", entry["b"])
	}
	if entry["c"] != 3 {
		t.Errorf("expected c=3 from entry, got %v", entry["c"])
	}
}

func TestInheritance_InvalidInheritType(t *testing.T) {
	tree := map[string]any{
		"_templates": map[string]any{
			"tmpl": map[string]any{"a": 1},
		},
		"entry": map[string]any{
			"_inherit": 42, // invalid: not string or list
		},
	}

	_, err := domain.ResolveInheritance(tree, "_templates", "_inherit")
	if err == nil {
		t.Fatal("expected error for invalid _inherit type")
	}
}

func TestInheritance_ListObjectMissingTemplate(t *testing.T) {
	tree := map[string]any{
		"entry": map[string]any{
			"_inherit": []any{
				map[string]any{
					// missing "template" key
					"except": []any{"key"},
				},
			},
		},
	}

	_, err := domain.ResolveInheritance(tree, "_templates", "_inherit")
	if err == nil {
		t.Fatal("expected error for list object missing template key")
	}
}
