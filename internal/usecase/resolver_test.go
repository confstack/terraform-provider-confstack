package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/filesystem"
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/logging"
	tmplAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/template"
	yamlAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/yaml"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/confstack/terraform-provider-confstack/internal/usecase"
)

func newResolver() *usecase.Resolver {
	return usecase.NewResolver(
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
		filesystem.NewExpander(),
	)
}

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolver_BasicSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "base.yaml", `
tags:
  managed_by: opentofu
`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	tags, ok := result.Output["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags map, got %T", result.Output["tags"])
	}
	if tags["managed_by"] != "opentofu" {
		t.Errorf("expected managed_by=opentofu, got %v", tags["managed_by"])
	}
	if len(result.LoadedLayers) != 1 {
		t.Errorf("expected 1 loaded layer, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_MergeOrder(t *testing.T) {
	dir := t.TempDir()
	base := writeTestFile(t, dir, "base.yaml", `
tags:
  managed_by: opentofu
  team: platform
eks:
  node_size: t3.medium
  min_nodes: 2
`)
	override := writeTestFile(t, dir, "override.yaml", `
tags:
  environment: prod
eks:
  node_size: m5.xlarge
  min_nodes: 3
`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{base, override})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	tags := result.Output["tags"].(map[string]any)
	if tags["managed_by"] != "opentofu" {
		t.Errorf("tags.managed_by: expected opentofu, got %v", tags["managed_by"])
	}
	if tags["team"] != "platform" {
		t.Errorf("tags.team: expected platform, got %v", tags["team"])
	}
	if tags["environment"] != "prod" {
		t.Errorf("tags.environment: expected prod, got %v", tags["environment"])
	}

	eks := result.Output["eks"].(map[string]any)
	if eks["node_size"] != "m5.xlarge" {
		t.Errorf("eks.node_size: expected m5.xlarge (last wins), got %v", eks["node_size"])
	}
	if eks["min_nodes"] != 3 {
		t.Errorf("eks.min_nodes: expected 3 (last wins), got %v", eks["min_nodes"])
	}

	if len(result.LoadedLayers) != 2 {
		t.Errorf("expected 2 loaded layers, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_WithTemplatingAndSecrets(t *testing.T) {
	dir := t.TempDir()
	base := writeTestFile(t, dir, "base.yaml", `
databases:
  main:
    host: db.example.internal
    password: {{ secret "DB_PASSWORD" }}
    vpc_id: {{ var "VPC_ID" }}
    engine: postgres
`)

	r := newResolver()
	req, err := domain.NewResolveRequest(
		[]string{base},
		domain.WithVariables(map[string]string{"VPC_ID": "vpc-12345678"}),
		domain.WithSecrets(map[string]string{"DB_PASSWORD": "super-secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	db := result.Output["databases"].(map[string]any)
	main := db["main"].(map[string]any)

	if main["password"] != "(sensitive)" {
		t.Errorf("expected (sensitive) in output, got %v", main["password"])
	}
	if main["vpc_id"] != "vpc-12345678" {
		t.Errorf("expected vpc-12345678, got %v", main["vpc_id"])
	}

	sensitiveDB := result.SensitiveOutput["databases"].(map[string]any)
	sensitiveMain := sensitiveDB["main"].(map[string]any)
	if sensitiveMain["password"] != "super-secret" {
		t.Errorf("sensitive_output: expected super-secret, got %v", sensitiveMain["password"])
	}

	if !result.SecretPaths["databases.main.password"] {
		t.Error("expected databases.main.password in secret paths")
	}
}

func TestResolver_NullTombstone(t *testing.T) {
	dir := t.TempDir()
	base := writeTestFile(t, dir, "base.yaml", `
key_to_delete: some_value
keep: yes
`)
	override := writeTestFile(t, dir, "override.yaml", `
key_to_delete: ~
`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{base, override})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := result.Output["key_to_delete"]; exists {
		t.Error("expected key_to_delete to be removed by null tombstone")
	}
	if result.Output["keep"] != "yes" {
		t.Error("expected keep=yes to remain")
	}
}

func TestResolver_MultiDoc(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
a: 1
---
b: 2
---
a: 3
`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if result.Output["a"] != 3 {
		t.Errorf("expected a=3 (last doc wins), got %v", result.Output["a"])
	}
	if result.Output["b"] != 2 {
		t.Errorf("expected b=2, got %v", result.Output["b"])
	}
}

func TestResolver_MissingLayer_Error(t *testing.T) {
	r := newResolver()
	req, err := domain.NewResolveRequest([]string{"/nonexistent/path/abc123.yaml"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent layer")
	}
}

func TestResolver_MissingLayer_Skip(t *testing.T) {
	dir := t.TempDir()
	existing := writeTestFile(t, dir, "base.yaml", `foo: bar`)

	r := newResolver()
	req, err := domain.NewResolveRequest(
		[]string{existing, "/nonexistent/missing.yaml"},
		domain.WithOnMissingLayer("skip"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", result.Output["foo"])
	}
	if len(result.LoadedLayers) != 1 {
		t.Errorf("expected 1 loaded layer, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_MissingLayer_Warn(t *testing.T) {
	dir := t.TempDir()
	existing := writeTestFile(t, dir, "base.yaml", `foo: bar`)

	r := newResolver()
	req, err := domain.NewResolveRequest(
		[]string{existing, "/nonexistent/missing.yaml"},
		domain.WithOnMissingLayer("warn"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", result.Output["foo"])
	}
}

func TestResolver_InvalidMissingLayer_RejectAtConstruction(t *testing.T) {
	// NewResolveRequest itself should reject invalid on_missing_layer before Resolve is called.
	_, err := domain.NewResolveRequest(
		[]string{"/any/path.yaml"},
		domain.WithOnMissingLayer("invalid-value"),
	)
	if err == nil {
		t.Fatal("expected error for invalid on_missing_layer from NewResolveRequest")
	}
}

func TestResolver_TemplatingError(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "base.yaml", `val: {{ var "MISSING_VAR_XYZ_123" }}`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing template variable")
	}
}

func TestResolver_YAMLParseError(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "base.yaml", "key: {\ninvalid yaml {{")

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for YAML parse error")
	}
}

func TestResolver_TypeMismatchError(t *testing.T) {
	dir := t.TempDir()
	base := writeTestFile(t, dir, "base.yaml", "key:\n  nested: value")
	override := writeTestFile(t, dir, "override.yaml", "key: scalar")

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{base, override})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestResolver_FlatOutput(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "base.yaml", `
database:
  host: localhost
  port: 5432
app: myapp
`)

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if result.FlatOutput["database.host"] != "localhost" {
		t.Errorf("expected database.host=localhost, got %v", result.FlatOutput["database.host"])
	}
	if result.FlatOutput["database.port"] != 5432 {
		t.Errorf("expected database.port=5432, got %v", result.FlatOutput["database.port"])
	}
	if result.FlatOutput["app"] != "myapp" {
		t.Errorf("expected app=myapp, got %v", result.FlatOutput["app"])
	}
}

func TestResolver_GlobPattern_SingleMatch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.yaml", "env: staging\n")

	r := newResolver()
	pattern := filepath.Join(dir, "*.yaml")
	req, err := domain.NewResolveRequest([]string{pattern})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["env"] != "staging" {
		t.Errorf("expected env=staging, got %v", result.Output["env"])
	}
	if len(result.LoadedLayers) != 1 {
		t.Errorf("expected 1 loaded layer, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_GlobPattern_AlphabeticOrder(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "01-base.yaml", "priority: low\nbase: true\n")
	writeTestFile(t, dir, "02-override.yaml", "priority: high\n")
	writeTestFile(t, dir, "03-final.yaml", "priority: final\n")

	r := newResolver()
	pattern := filepath.Join(dir, "*.yaml")
	req, err := domain.NewResolveRequest([]string{pattern})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Last alphabetically wins: 03-final.yaml
	if result.Output["priority"] != "final" {
		t.Errorf("expected priority=final (last file wins), got %v", result.Output["priority"])
	}
	// base key from 01-base.yaml should be present
	if result.Output["base"] != true {
		t.Errorf("expected base=true from first file, got %v", result.Output["base"])
	}
	if len(result.LoadedLayers) != 3 {
		t.Errorf("expected 3 loaded layers, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_GlobPattern_MixedLiteralAndGlob(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.yaml", "env: base\nbase_key: base_val\n")
	overridesDir := filepath.Join(dir, "overrides")
	writeTestFile(t, overridesDir, "01-net.yaml", "env: net\nnet_key: net_val\n")
	writeTestFile(t, overridesDir, "02-compute.yaml", "env: compute\ncompute_key: compute_val\n")

	r := newResolver()
	basePath := filepath.Join(dir, "base.yaml")
	globPattern := filepath.Join(overridesDir, "*.yaml")
	req, err := domain.NewResolveRequest([]string{basePath, globPattern})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Last glob match wins overall: 02-compute.yaml
	if result.Output["env"] != "compute" {
		t.Errorf("expected env=compute (last layer wins), got %v", result.Output["env"])
	}
	if result.Output["base_key"] != "base_val" {
		t.Errorf("expected base_key=base_val, got %v", result.Output["base_key"])
	}
	if result.Output["net_key"] != "net_val" {
		t.Errorf("expected net_key=net_val, got %v", result.Output["net_key"])
	}
	if result.Output["compute_key"] != "compute_val" {
		t.Errorf("expected compute_key=compute_val, got %v", result.Output["compute_key"])
	}
	if len(result.LoadedLayers) != 3 {
		t.Errorf("expected 3 loaded layers, got %d", len(result.LoadedLayers))
	}
}

func TestResolver_GlobPattern_Doublestar(t *testing.T) {
	dir := t.TempDir()
	subA := filepath.Join(dir, "a")
	subB := filepath.Join(dir, "b")
	writeTestFile(t, subA, "config.yaml", "from: a\n")
	writeTestFile(t, subB, "config.yaml", "from: b\n")

	r := newResolver()
	pattern := filepath.Join(dir, "**", "*.yaml")
	req, err := domain.NewResolveRequest([]string{pattern})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.LoadedLayers) != 2 {
		t.Errorf("expected 2 loaded layers, got %d: %v", len(result.LoadedLayers), result.LoadedLayers)
	}
}

func TestResolver_GlobPattern_NoMatch_Error(t *testing.T) {
	dir := t.TempDir()
	r := newResolver()
	pattern := filepath.Join(dir, "*.yaml")
	req, err := domain.NewResolveRequest([]string{pattern},
		domain.WithOnMissingLayer("error"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unmatched glob with on_missing_layer=error")
	}
	var globErr *domain.NoGlobMatchError
	if !errors.As(err, &globErr) {
		t.Errorf("expected NoGlobMatchError, got %T: %v", err, err)
	}
}

func TestResolver_GlobPattern_NoMatch_Skip(t *testing.T) {
	dir := t.TempDir()
	litPath := writeTestFile(t, dir, "base.yaml", "env: present\n")

	r := newResolver()
	pattern := filepath.Join(dir, "nonexistent", "*.yaml")
	req, err := domain.NewResolveRequest([]string{litPath, pattern},
		domain.WithOnMissingLayer("skip"),
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with on_missing_layer=skip: %v", err)
	}
	if result.Output["env"] != "present" {
		t.Errorf("expected env=present from literal layer, got %v", result.Output["env"])
	}
	if len(result.LoadedLayers) != 1 {
		t.Errorf("expected only 1 loaded layer (glob skipped), got %d", len(result.LoadedLayers))
	}
}

func TestResolver_LiteralPrefix_LoadsBracketedFilename(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config[prod].yaml", "env: prod\n")

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{domain.LiteralLayerPrefix + path})
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["env"] != "prod" {
		t.Errorf("expected env=prod, got %v", result.Output["env"])
	}
	if len(result.LoadedLayers) != 1 || result.LoadedLayers[0] != path {
		t.Errorf("expected loaded layer %q, got %v", path, result.LoadedLayers)
	}
}

func TestResolver_LiteralPrefix_MissingFile_RespectsOnMissingLayer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config[prod].yaml")

	r := newResolver()
	req, err := domain.NewResolveRequest([]string{domain.LiteralLayerPrefix + path}, domain.WithOnMissingLayer("skip"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with on_missing_layer=skip: %v", err)
	}
	if len(result.LoadedLayers) != 0 {
		t.Errorf("expected no loaded layers, got %v", result.LoadedLayers)
	}
}
