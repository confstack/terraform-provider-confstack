package template_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tmplAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/template"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// newReq creates a minimal ResolveRequest for testing.
func newReq() domain.ResolveRequest {
	req, _ := domain.NewResolveRequest([]string{"test.yaml"})
	return req
}

func TestEngine_VarFromMap(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()
	req.Variables = map[string]string{"VPC_ID": "vpc-12345"}

	out, _, err := e.Process(context.Background(), []byte(`vpc: {{ var "VPC_ID" }}`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `vpc-12345`) || strings.Contains(string(out), `"vpc-12345"`) {
		t.Errorf("expected unquoted vpc-12345 in output, got %s", string(out))
	}
}

func TestEngine_VarFromEnv(t *testing.T) {
	if err := os.Setenv("TEST_VAR_12345", "env-value"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_VAR_12345"); err != nil {
			t.Errorf("failed to unset env var: %v", err)
		}
	}()

	e := tmplAdapter.NewEngine()
	req := newReq()

	out, _, err := e.Process(context.Background(), []byte(`val: {{ var "TEST_VAR_12345" }}`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `env-value`) || strings.Contains(string(out), `"env-value"`) {
		t.Errorf("expected unquoted env-value in output, got %s", string(out))
	}
}

func TestEngine_VarMissing(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()

	_, _, err := e.Process(context.Background(), []byte(`{{ var "DEFINITELY_NOT_SET_XYZ" }}`), "test.yaml", req, "nonce1")
	if err == nil {
		t.Fatal("expected error for missing var")
	}
	var mve *domain.MissingVariableError
	if !errors.As(err, &mve) {
		t.Errorf("expected MissingVariableError, got %T: %v", err, err)
	}
}

func TestEngine_SecretFromMap(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()
	req.Secrets = map[string]string{"DB_PASS": "s3cr3t"}

	out, sentinels, err := e.Process(context.Background(), []byte(`password: {{ secret "DB_PASS" }}`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}

	outStr := string(out)
	if !strings.Contains(outStr, "__CONFSTACK_SECRET_") {
		t.Errorf("expected sentinel pattern in output, got %s", outStr)
	}
	if len(sentinels) != 1 {
		t.Errorf("expected 1 sentinel, got %d", len(sentinels))
	}
	for sentinel, val := range sentinels {
		if val != "s3cr3t" {
			t.Errorf("expected sentinel value s3cr3t, got %s", val)
		}
		if !strings.HasPrefix(sentinel, "__CONFSTACK_SECRET_") {
			t.Errorf("expected sentinel prefix, got %s", sentinel)
		}
	}
}

func TestEngine_SecretMissing(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()

	_, _, err := e.Process(context.Background(), []byte(`{{ secret "MISSING_SECRET_XYZ" }}`), "test.yaml", req, "nonce1")
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
	var mve *domain.MissingVariableError
	if !errors.As(err, &mve) {
		t.Errorf("expected MissingVariableError, got %T: %v", err, err)
	}
}

func TestEngine_SprigFunctions(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()

	out, _, err := e.Process(context.Background(), []byte(`value: {{ "hello" | upper }}`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "HELLO") {
		t.Errorf("expected HELLO from Sprig upper, got %s", string(out))
	}
}

func TestEngine_VarInlineConcatenation(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()
	req.Variables = map[string]string{"HOST": "myhost", "PORT": "8080"}

	out, _, err := e.Process(context.Background(), []byte(`url: http://{{ var "HOST" }}:{{ var "PORT" }}/api`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "http://myhost:8080/api") {
		t.Errorf("expected http://myhost:8080/api in output, got %s", outStr)
	}
}

func TestEngine_SecretInlineConcatenation(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()
	req.Secrets = map[string]string{"PASS": "s3cr3t"}

	out, sentinels, err := e.Process(context.Background(), []byte(`dsn: postgres://user:{{ secret "PASS" }}@host/db`), "test.yaml", req, "nonce1")
	if err != nil {
		t.Fatal(err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "postgres://user:") || !strings.Contains(outStr, "@host/db") {
		t.Errorf("expected inline DSN with sentinel in output, got %s", outStr)
	}
	if !strings.Contains(outStr, "__CONFSTACK_SECRET_") {
		t.Errorf("expected sentinel embedded in DSN, got %s", outStr)
	}
	if len(sentinels) != 1 {
		t.Errorf("expected 1 sentinel in map, got %d", len(sentinels))
	}
}

func TestEngine_SentinelDifferentNonces(t *testing.T) {
	e := tmplAdapter.NewEngine()
	req := newReq()
	req.Secrets = map[string]string{"KEY": "value"}

	_, s1, _ := e.Process(context.Background(), []byte(`{{ secret "KEY" }}`), "test.yaml", req, "nonce-a")
	_, s2, _ := e.Process(context.Background(), []byte(`{{ secret "KEY" }}`), "test.yaml", req, "nonce-b")

	var sent1, sent2 string
	for k := range s1 {
		sent1 = k
	}
	for k := range s2 {
		sent2 = k
	}
	if sent1 == sent2 {
		t.Error("expected different sentinels for different nonces")
	}
}
