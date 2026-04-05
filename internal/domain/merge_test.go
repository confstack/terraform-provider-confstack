package domain_test

import (
	"errors"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestDeepMerge_MapOnMap(t *testing.T) {
	base := map[string]any{"a": map[string]any{"x": 1, "y": 2}}
	overlay := map[string]any{"a": map[string]any{"y": 99, "z": 3}}
	result, err := domain.DeepMerge(base, overlay, "", "base.yaml", "overlay.yaml")
	if err != nil {
		t.Fatal(err)
	}
	inner := result["a"].(map[string]any)
	if inner["x"] != 1 {
		t.Error("expected x=1 to be preserved")
	}
	if inner["y"] != 99 {
		t.Error("expected y=99 from overlay")
	}
	if inner["z"] != 3 {
		t.Error("expected z=3 from overlay")
	}
}

func TestDeepMerge_ScalarReplace(t *testing.T) {
	base := map[string]any{"a": "foo"}
	overlay := map[string]any{"a": "bar"}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	if result["a"] != "bar" {
		t.Errorf("expected a=bar, got %v", result["a"])
	}
}

func TestDeepMerge_ListReplace(t *testing.T) {
	base := map[string]any{"a": []any{1, 2, 3}}
	overlay := map[string]any{"a": []any{4, 5}}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	list := result["a"].([]any)
	if len(list) != 2 || list[0] != 4 {
		t.Errorf("expected list [4, 5], got %v", list)
	}
}

func TestDeepMerge_NullTombstone(t *testing.T) {
	base := map[string]any{"a": "value", "b": "keep"}
	overlay := map[string]any{"a": nil}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := result["a"]; exists {
		t.Error("expected key 'a' to be deleted by null tombstone")
	}
	if result["b"] != "keep" {
		t.Error("expected key 'b' to be preserved")
	}
}

func TestDeepMerge_TypeMismatch_MapVsScalar(t *testing.T) {
	base := map[string]any{"a": map[string]any{"x": 1}}
	overlay := map[string]any{"a": "scalar"}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error on map vs scalar mismatch")
	}
	var mce *domain.MergeConflictError
	if !errors.As(err, &mce) {
		t.Errorf("expected MergeConflictError, got %T: %v", err, err)
	}
}

func TestDeepMerge_TypeMismatch_ScalarVsMap(t *testing.T) {
	base := map[string]any{"a": "scalar"}
	overlay := map[string]any{"a": map[string]any{"x": 1}}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error on scalar vs map mismatch")
	}
}

func TestDeepMerge_TypeMismatch_MapVsList(t *testing.T) {
	base := map[string]any{"a": map[string]any{"x": 1}}
	overlay := map[string]any{"a": []any{1, 2}}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error on map vs list mismatch")
	}
}

func TestDeepMerge_NullBase(t *testing.T) {
	base := map[string]any{"a": nil}
	overlay := map[string]any{"a": map[string]any{"x": 1}}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	inner, ok := result["a"].(map[string]any)
	if !ok || inner["x"] != 1 {
		t.Errorf("expected a={x:1}, got %v", result["a"])
	}
}

func TestDeepMerge_DeepNesting(t *testing.T) {
	base := map[string]any{
		"l1": map[string]any{
			"l2": map[string]any{
				"l3": map[string]any{
					"l4": map[string]any{
						"l5": "original",
					},
				},
			},
		},
	}
	overlay := map[string]any{
		"l1": map[string]any{
			"l2": map[string]any{
				"l3": map[string]any{
					"l4": map[string]any{
						"l5": "overridden",
					},
				},
			},
		},
	}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	l5 := result["l1"].(map[string]any)["l2"].(map[string]any)["l3"].(map[string]any)["l4"].(map[string]any)["l5"]
	if l5 != "overridden" {
		t.Errorf("expected l5=overridden, got %v", l5)
	}
}

func TestMergeAll_PriorityOrder(t *testing.T) {
	files := []domain.DiscoveredFile{
		{Path: "p1", Priority: 1},
		{Path: "p3", Priority: 3},
		{Path: "p5", Priority: 5},
		{Path: "p8", Priority: 8},
	}
	data := map[string][]map[string]any{
		"p1": {{"key": "p1_value", "p1_only": "yes"}},
		"p3": {{"key": "p3_value"}},
		"p5": {{"key": "p5_value"}},
		"p8": {{"key": "p8_value", "p8_only": "yes"}},
	}
	result, err := domain.MergeAll(files, data)
	if err != nil {
		t.Fatal(err)
	}
	if result["key"] != "p8_value" {
		t.Errorf("expected key=p8_value (highest priority), got %v", result["key"])
	}
	if result["p1_only"] != "yes" {
		t.Error("expected p1_only from lowest priority to be preserved")
	}
	if result["p8_only"] != "yes" {
		t.Error("expected p8_only from highest priority to be present")
	}
}

func TestDeepMerge_BoolVsBool(t *testing.T) {
	// Covers bool type in typeName (via scalar replace branch, no error)
	base := map[string]any{"flag": true}
	overlay := map[string]any{"flag": false}
	result, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err != nil {
		t.Fatal(err)
	}
	if result["flag"] != false {
		t.Error("expected flag=false after scalar replace")
	}
}

func TestDeepMerge_BoolVsMap(t *testing.T) {
	// Covers bool going into typeName error path
	base := map[string]any{"flag": true}
	overlay := map[string]any{"flag": map[string]any{"x": 1}}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error: bool vs map mismatch")
	}
}

func TestDeepMerge_NumberVsMap(t *testing.T) {
	// Covers number going into typeName error path
	base := map[string]any{"n": 42}
	overlay := map[string]any{"n": map[string]any{"x": 1}}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error: number vs map mismatch")
	}
}

func TestDeepMerge_AllTypeNames(t *testing.T) {
	// Exercise typeName with list type mismatch
	base := map[string]any{"a": []any{1, 2}}
	overlay := map[string]any{"a": map[string]any{"x": 1}}
	_, err := domain.DeepMerge(base, overlay, "", "b", "o")
	if err == nil {
		t.Fatal("expected error on list vs map mismatch")
	}
}

func TestMergeAll_EmptyFileData(t *testing.T) {
	files := []domain.DiscoveredFile{
		{Path: "p1", Priority: 1},
	}
	// p1 has no data (not present in map)
	result, err := domain.MergeAll(files, map[string][]map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}
