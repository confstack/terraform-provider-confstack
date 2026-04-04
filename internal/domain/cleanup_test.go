package domain_test

import (
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestStripReservedKeys_TopLevel(t *testing.T) {
	tree := map[string]any{
		"_templates": map[string]any{"base": map[string]any{"x": 1}},
		"_inherit":   "base",
		"keep":       "yes",
	}
	result := domain.StripReservedKeys(tree, "_templates", "_inherit")
	if _, ok := result["_templates"]; ok {
		t.Error("expected _templates to be removed")
	}
	if _, ok := result["_inherit"]; ok {
		t.Error("expected _inherit to be removed")
	}
	if result["keep"] != "yes" {
		t.Error("expected keep=yes to remain")
	}
}

func TestStripReservedKeys_Nested(t *testing.T) {
	tree := map[string]any{
		"sqs": map[string]any{
			"_templates": map[string]any{"t": map[string]any{}},
			"queue1": map[string]any{
				"_inherit": "t",
				"x":        1,
			},
		},
	}
	result := domain.StripReservedKeys(tree, "_templates", "_inherit")
	sqs := result["sqs"].(map[string]any)
	if _, ok := sqs["_templates"]; ok {
		t.Error("expected nested _templates to be removed")
	}
	q1 := sqs["queue1"].(map[string]any)
	if _, ok := q1["_inherit"]; ok {
		t.Error("expected nested _inherit to be removed")
	}
	if q1["x"] != 1 {
		t.Error("expected x=1 to remain")
	}
}
