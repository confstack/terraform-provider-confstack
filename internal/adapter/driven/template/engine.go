package template

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

// Engine implements port/output.TemplateEngine using text/template + Sprig.
type Engine struct{}

// NewEngine returns a new Go template Engine with Sprig functions included.
func NewEngine() *Engine {
	return &Engine{}
}

// Process renders data as a Go template with var/secret injection.
// Returns processed bytes and a sentinel→realValue map for secrets.
func (e *Engine) Process(ctx context.Context, data []byte, filePath string, req domain.ResolveRequest, nonce string) ([]byte, map[string]string, error) {
	sentinelMap := make(map[string]string)

	funcMap := sprig.TxtFuncMap()

	// var(key): looks up from variables map then env, outputs JSON-encoded string for YAML safety
	funcMap["var"] = func(key string) (string, error) {
		if v, ok := req.Variables[key]; ok {
			return jsonEncode(v)
		}
		if v := os.Getenv(key); v != "" {
			return jsonEncode(v)
		}
		return "", &domain.MissingVariableError{Key: key, FuncName: "var"}
	}

	// secret(key): looks up from secrets map then env, returns a sentinel string
	funcMap["secret"] = func(key string) (string, error) {
		var realValue string
		if v, ok := req.Secrets[key]; ok {
			realValue = v
		} else if v := os.Getenv(key); v != "" {
			realValue = v
		} else {
			return "", &domain.MissingVariableError{Key: key, FuncName: "secret"}
		}

		sentinel := makeSentinel(nonce, key)
		sentinelMap[sentinel] = realValue
		return sentinel, nil
	}

	tmpl, err := template.New(filePath).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, nil, &domain.TemplateRenderError{FilePath: filePath, Detail: "parse", Cause: err}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, nil, &domain.TemplateRenderError{FilePath: filePath, Detail: "execution", Cause: err}
	}

	return buf.Bytes(), sentinelMap, nil
}

// makeSentinel creates a deterministic sentinel string for a given nonce+key combination.
func makeSentinel(nonce, key string) string {
	h := sha256.Sum256([]byte(nonce + key))
	return fmt.Sprintf("__CONFSTACK_SECRET_%x__", h)
}

// jsonEncode encodes a string as a JSON string for safe YAML injection.
// The result is a double-quoted string that is valid in both JSON and YAML scalar contexts.
func jsonEncode(v string) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encoding variable value: %w", err)
	}
	return string(b), nil
}
