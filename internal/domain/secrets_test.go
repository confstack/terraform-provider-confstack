package domain_test

import (
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestResolveSecrets_Basic(t *testing.T) {
	sentinel := "__CONFSTACK_SECRET_abc123__"
	tree := map[string]any{
		"db": map[string]any{
			"password": sentinel,
			"host":     "localhost",
		},
	}
	sentinelMap := map[string]string{sentinel: "super-secret"}

	redacted, full, paths, err := domain.ResolveSecrets(tree, sentinelMap)
	if err != nil {
		t.Fatal(err)
	}

	dbRedacted := redacted["db"].(map[string]any)
	if dbRedacted["password"] != "(sensitive)" {
		t.Errorf("expected redacted password=(sensitive), got %v", dbRedacted["password"])
	}
	if dbRedacted["host"] != "localhost" {
		t.Error("expected host to remain unchanged")
	}

	dbFull := full["db"].(map[string]any)
	if dbFull["password"] != "super-secret" {
		t.Errorf("expected full password=super-secret, got %v", dbFull["password"])
	}

	if !paths["db.password"] {
		t.Error("expected db.password to be in secret paths")
	}
	if paths["db.host"] {
		t.Error("expected db.host NOT to be in secret paths")
	}
}

func TestResolveSecrets_NonSentinel(t *testing.T) {
	tree := map[string]any{"key": "plain_value"}
	redacted, full, paths, err := domain.ResolveSecrets(tree, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if redacted["key"] != "plain_value" || full["key"] != "plain_value" {
		t.Error("expected plain_value unchanged in both outputs")
	}
	if len(paths) != 0 {
		t.Error("expected no secret paths for non-sentinel values")
	}
}

func TestResolveSecrets_NestedPaths(t *testing.T) {
	s1 := "__CONFSTACK_SECRET_s1__"
	s2 := "__CONFSTACK_SECRET_s2__"
	tree := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": s1,
			},
		},
		"x": s2,
	}
	sentinelMap := map[string]string{s1: "val1", s2: "val2"}

	_, _, paths, err := domain.ResolveSecrets(tree, sentinelMap)
	if err != nil {
		t.Fatal(err)
	}
	if !paths["a.b.c"] {
		t.Error("expected a.b.c to be in secret paths")
	}
	if !paths["x"] {
		t.Error("expected x to be in secret paths")
	}
}

func TestResolveSecrets_ListWithSentinel(t *testing.T) {
	s1 := "__CONFSTACK_SECRET_aaa__"
	tree := map[string]any{
		"items": []any{"plain", s1, 42},
	}
	sentinelMap := map[string]string{s1: "secret-val"}

	redacted, full, paths, err := domain.ResolveSecrets(tree, sentinelMap)
	if err != nil {
		t.Fatal(err)
	}

	redItems := redacted["items"].([]any)
	if redItems[0] != "plain" {
		t.Errorf("expected items[0]=plain, got %v", redItems[0])
	}
	if redItems[1] != "(sensitive)" {
		t.Errorf("expected items[1]=(sensitive), got %v", redItems[1])
	}
	if redItems[2] != 42 {
		t.Errorf("expected items[2]=42, got %v", redItems[2])
	}

	fullItems := full["items"].([]any)
	if fullItems[1] != "secret-val" {
		t.Errorf("expected full items[1]=secret-val, got %v", fullItems[1])
	}

	if !paths["items[1]"] {
		t.Error("expected items[1] to be in secret paths")
	}
}

func TestResolveSecrets_UnknownSentinelNotInMap(t *testing.T) {
	// A sentinel that doesn't appear in sentinelMap (shouldn't happen but covers the fallback branch)
	sentinel := "__CONFSTACK_SECRET_unknown__"
	tree := map[string]any{"key": sentinel}
	// Empty sentinel map - sentinel won't be found
	_, full, _, err := domain.ResolveSecrets(tree, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	// Fallback: returns the sentinel string itself
	if full["key"] != sentinel {
		t.Errorf("expected sentinel passthrough, got %v", full["key"])
	}
}
