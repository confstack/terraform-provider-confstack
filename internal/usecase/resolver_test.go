package usecase_test

import (
	"context"
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
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
	)
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolver_BasicSingleFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "_global"), "defaults.common.yaml", `
tags:
  managed_by: opentofu
`)

	r := newResolver()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "dev"

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
	if len(result.LoadedFiles) != 1 {
		t.Errorf("expected 1 loaded file, got %d", len(result.LoadedFiles))
	}
}

func TestResolver_Section12_ProdAcme(t *testing.T) {
	dir := t.TempDir()

	// _global/defaults.common.yaml
	writeTestFile(t, filepath.Join(dir, "_global"), "defaults.common.yaml", `
tags:
  managed_by: opentofu
  team: platform

sqs_queues:
  _templates:
    sqs_base:
      retention: 86400
      dlq: true
      visibility_timeout: 30

dynamodb_tables:
  _templates:
    dynamo_base:
      billing_mode: PAY_PER_REQUEST

s3_buckets:
  _templates:
    s3_base:
      versioning: true
      encryption: AES256
`)

	// _global/databases.common.yaml
	writeTestFile(t, filepath.Join(dir, "_global"), "databases.common.yaml", `
databases:
  main:
    host: db.example.internal
    password: {{ secret "DB_PASSWORD" }}
    vpc_id: {{ var "VPC_ID" }}
    engine: postgres
`)

	// _global/compute.common.yaml
	writeTestFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", `
eks:
  node_size: t3.medium
  min_nodes: 2
  max_nodes: 10

sqs_queues:
  _templates:
    standard:
      retention: 86400
      dlq: true
      visibility_timeout: 30
    high_retention:
      retention: 604800
      dlq: true
      visibility_timeout: 30
    critical:
      retention: 604800
      dlq: true
      visibility_timeout: 30
      dlq_max_retries: 5

  orders:
    _inherit:
      - template: standard
        except:
          - dlq
      - template: critical
  notifications:
    _inherit: standard
`)

	// prod/defaults.common.yaml
	writeTestFile(t, filepath.Join(dir, "prod"), "defaults.common.yaml", `
tags:
  environment: prod
`)

	// prod/compute.acme.yaml
	writeTestFile(t, filepath.Join(dir, "prod"), "compute.acme.yaml", `
eks:
  node_size: m5.xlarge
  min_nodes: 3
  max_nodes: 50

sqs_queues:
  orders:
    visibility_timeout: 120
  payments:
    _inherit: critical
    retention: 86400
`)

	r := newResolver()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"
	req.Tenant = "acme"
	req.Variables = map[string]string{"VPC_ID": "vpc-12345678"}
	req.Secrets = map[string]string{"DB_PASSWORD": "super-secret"}

	result, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	out := result.Output

	// Check tags (merged from global defaults + prod defaults)
	tags := out["tags"].(map[string]any)
	if tags["managed_by"] != "opentofu" {
		t.Errorf("tags.managed_by: expected opentofu, got %v", tags["managed_by"])
	}
	if tags["team"] != "platform" {
		t.Errorf("tags.team: expected platform, got %v", tags["team"])
	}
	if tags["environment"] != "prod" {
		t.Errorf("tags.environment: expected prod, got %v", tags["environment"])
	}

	// Check EKS (prod/acme overrides global)
	eks := out["eks"].(map[string]any)
	if eks["node_size"] != "m5.xlarge" {
		t.Errorf("eks.node_size: expected m5.xlarge, got %v", eks["node_size"])
	}
	if eks["min_nodes"] != 3 {
		t.Errorf("eks.min_nodes: expected 3, got %v", eks["min_nodes"])
	}
	if eks["max_nodes"] != 50 {
		t.Errorf("eks.max_nodes: expected 50, got %v", eks["max_nodes"])
	}

	// Check sqs_queues
	sqs := out["sqs_queues"].(map[string]any)

	// orders: standard (except dlq) + critical, then prod/acme visibility_timeout=120
	orders := sqs["orders"].(map[string]any)
	if orders["retention"] != 604800 {
		t.Errorf("orders.retention: expected 604800 from critical, got %v", orders["retention"])
	}
	if orders["visibility_timeout"] != 120 {
		t.Errorf("orders.visibility_timeout: expected 120 from prod override, got %v", orders["visibility_timeout"])
	}
	if orders["dlq"] != true {
		t.Errorf("orders.dlq: expected true, got %v", orders["dlq"])
	}
	if orders["dlq_max_retries"] != 5 {
		t.Errorf("orders.dlq_max_retries: expected 5, got %v", orders["dlq_max_retries"])
	}

	// notifications: standard template
	notif := sqs["notifications"].(map[string]any)
	if notif["retention"] != 86400 {
		t.Errorf("notifications.retention: expected 86400, got %v", notif["retention"])
	}
	if notif["dlq"] != true {
		t.Errorf("notifications.dlq: expected true, got %v", notif["dlq"])
	}

	// payments: critical template with retention=86400 override
	payments := sqs["payments"].(map[string]any)
	if payments["retention"] != 86400 {
		t.Errorf("payments.retention: expected 86400 (entry override), got %v", payments["retention"])
	}
	if payments["dlq"] != true {
		t.Errorf("payments.dlq: expected true from critical, got %v", payments["dlq"])
	}
	if payments["dlq_max_retries"] != 5 {
		t.Errorf("payments.dlq_max_retries: expected 5 from critical, got %v", payments["dlq_max_retries"])
	}

	// databases: password should be redacted in output
	db := out["databases"].(map[string]any)
	main := db["main"].(map[string]any)
	if main["password"] != "(sensitive)" {
		t.Errorf("databases.main.password: expected (sensitive) in output, got %v", main["password"])
	}
	if main["vpc_id"] != "vpc-12345678" {
		t.Errorf("databases.main.vpc_id: expected vpc-12345678, got %v", main["vpc_id"])
	}
	if main["host"] != "db.example.internal" {
		t.Errorf("databases.main.host: expected db.example.internal, got %v", main["host"])
	}

	// sensitive_output should have real password
	sensitiveDB := result.SensitiveOutput["databases"].(map[string]any)
	sensitiveMain := sensitiveDB["main"].(map[string]any)
	if sensitiveMain["password"] != "super-secret" {
		t.Errorf("sensitive_output.databases.main.password: expected super-secret, got %v", sensitiveMain["password"])
	}

	// secret paths
	if !result.SecretPaths["databases.main.password"] {
		t.Error("expected databases.main.password in secret paths")
	}

	// loaded files count: _global/defaults.common + _global/databases.common + _global/compute.common + prod/defaults.common + prod/compute.acme = 5
	if len(result.LoadedFiles) != 5 {
		t.Errorf("expected 5 loaded files, got %d: %v", len(result.LoadedFiles), result.LoadedFiles)
	}
}

func TestResolver_NullTombstone_Integration(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "_global"), "defaults.common.yaml", `
key_to_delete: some_value
keep: yes
`)
	writeTestFile(t, filepath.Join(dir, "prod"), "defaults.common.yaml", `
key_to_delete: ~
`)

	r := newResolver()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

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
	writeTestFile(t, filepath.Join(dir, "_global"), "config.common.yaml", `
a: 1
---
b: 2
---
a: 3
`)

	r := newResolver()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "dev"

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

func TestResolver_MissingConfigDir(t *testing.T) {
	r := usecase.NewResolver(
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
	)

	req := domain.DefaultResolveRequest()
	req.ConfigDir = "/nonexistent/path/abc123"
	req.Environment = "prod"

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent config_dir")
	}
}

func TestResolver_TemplatingError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_global"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a file with a missing var
	if err := os.WriteFile(filepath.Join(dir, "_global", "defaults.common.yaml"),
		[]byte(`val: {{ var "MISSING_VAR_XYZ_123" }}`), 0o644); err != nil {
		t.Fatal(err)
	}

	r := usecase.NewResolver(
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
	)

	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "dev"

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing template variable")
	}
}

func TestResolver_YAMLParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_global"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_global", "defaults.common.yaml"),
		[]byte("key: {\ninvalid yaml {{"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := usecase.NewResolver(
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
	)

	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "dev"

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for YAML parse error")
	}
}

func TestResolver_TypeMismatchError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_global"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_global", "defaults.common.yaml"),
		[]byte("key:\n  nested: value"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prod", "defaults.common.yaml"),
		[]byte("key: scalar"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := usecase.NewResolver(
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewNopLogger(),
	)

	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}
