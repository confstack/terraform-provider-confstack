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
