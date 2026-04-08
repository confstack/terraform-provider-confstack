package yaml_test

import (
	"context"
	"errors"
	"testing"

	yamlAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/yaml"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestParser_SingleDoc(t *testing.T) {
	p := yamlAdapter.NewParser()
	data := []byte(`
key: value
nested:
  a: 1
  b: 2
`)
	docs, err := p.ParseMultiDoc(context.Background(), data, "test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0]["key"] != "value" {
		t.Errorf("expected key=value, got %v", docs[0]["key"])
	}
	nested := docs[0]["nested"].(map[string]any)
	if nested["a"] != 1 {
		t.Errorf("expected a=1, got %v", nested["a"])
	}
}

func TestParser_MultiDoc(t *testing.T) {
	p := yamlAdapter.NewParser()
	data := []byte(`
key: value1
---
key: value2
other: yes
`)
	docs, err := p.ParseMultiDoc(context.Background(), data, "test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	if docs[0]["key"] != "value1" {
		t.Error("doc 0: expected key=value1")
	}
	if docs[1]["key"] != "value2" {
		t.Error("doc 1: expected key=value2")
	}
}

func TestParser_EmptyFile(t *testing.T) {
	p := yamlAdapter.NewParser()
	docs, err := p.ParseMultiDoc(context.Background(), []byte{}, "empty.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 empty doc, got %d", len(docs))
	}
	if len(docs[0]) != 0 {
		t.Error("expected empty map for empty file")
	}
}

func TestParser_SyntaxError(t *testing.T) {
	p := yamlAdapter.NewParser()
	data := []byte("key: {\ninvalid")
	_, err := p.ParseMultiDoc(context.Background(), data, "bad.yaml")
	if err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
	var pe *domain.ParseError
	if !errors.As(err, &pe) {
		t.Errorf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParser_ListValues(t *testing.T) {
	p := yamlAdapter.NewParser()
	data := []byte(`
items:
  - alpha
  - beta
  - 42
nested_list:
  - key: a
    val: 1
  - key: b
    val: 2
`)
	docs, err := p.ParseMultiDoc(context.Background(), data, "list.yaml")
	if err != nil {
		t.Fatal(err)
	}
	items, ok := docs[0]["items"].([]any)
	if !ok {
		t.Fatalf("expected items to be []any, got %T", docs[0]["items"])
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != "alpha" {
		t.Errorf("expected items[0]=alpha, got %v", items[0])
	}
	if items[2] != 42 {
		t.Errorf("expected items[2]=42, got %v", items[2])
	}
	// Nested list of maps exercises convertValue map[string]any recursion
	nested, ok := docs[0]["nested_list"].([]any)
	if !ok {
		t.Fatalf("expected nested_list to be []any, got %T", docs[0]["nested_list"])
	}
	first := nested[0].(map[string]any)
	if first["key"] != "a" {
		t.Errorf("expected nested_list[0].key=a, got %v", first["key"])
	}
}

func TestParser_NonMapTopLevelDoc(t *testing.T) {
	p := yamlAdapter.NewParser()
	// A top-level YAML list (not a map) should return a ParseError.
	data := []byte("- item1\n- item2\n")
	_, err := p.ParseMultiDoc(context.Background(), data, "list.yaml")
	if err == nil {
		t.Fatal("expected error for top-level list document")
	}
	var pe *domain.ParseError
	if !errors.As(err, &pe) {
		t.Errorf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParser_NullDocInMultiDoc(t *testing.T) {
	p := yamlAdapter.NewParser()
	// A null document (empty section between ---) should produce an empty map.
	data := []byte("key: value\n---\n---\nother: true\n")
	docs, err := p.ParseMultiDoc(context.Background(), data, "multidoc.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
	if len(docs[1]) != 0 {
		t.Errorf("expected middle doc to be empty map, got %v", docs[1])
	}
}

func TestParser_Anchors(t *testing.T) {
	p := yamlAdapter.NewParser()
	data := []byte(`
defaults: &defaults
  timeout: 30
  retry: 3

service:
  <<: *defaults
  name: my-service
`)
	docs, err := p.ParseMultiDoc(context.Background(), data, "anchors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	svc := docs[0]["service"].(map[string]any)
	if svc["timeout"] != 30 {
		t.Errorf("expected timeout=30 from anchor, got %v", svc["timeout"])
	}
	if svc["name"] != "my-service" {
		t.Errorf("expected name=my-service, got %v", svc["name"])
	}
}
