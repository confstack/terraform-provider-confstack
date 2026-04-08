package domain_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestNoGlobMatchError_Message(t *testing.T) {
	err := &domain.NoGlobMatchError{Pattern: "configs/**/*.yaml"}
	msg := err.Error()
	if !strings.Contains(msg, "configs/**/*.yaml") {
		t.Errorf("expected error message to contain pattern, got: %q", msg)
	}
	if !strings.Contains(msg, "matched no files") {
		t.Errorf("expected error message to mention 'matched no files', got: %q", msg)
	}
}

func TestParseError_ErrorAndUnwrap(t *testing.T) {
	cause := fmt.Errorf("underlying cause")
	err := &domain.ParseError{FilePath: "config.yaml", Detail: "bad syntax", Cause: cause}
	msg := err.Error()
	if !strings.Contains(msg, "config.yaml") {
		t.Errorf("expected file path in error message, got %q", msg)
	}
	if !strings.Contains(msg, "bad syntax") {
		t.Errorf("expected detail in error message, got %q", msg)
	}
	if !errors.Is(err, cause) {
		t.Error("expected Unwrap to return cause")
	}

	// Without cause
	errNoCause := &domain.ParseError{FilePath: "config.yaml", Detail: "bad syntax"}
	if !strings.Contains(errNoCause.Error(), "config.yaml") {
		t.Errorf("expected file path in error message, got %q", errNoCause.Error())
	}
}

func TestFileReadError_ErrorAndUnwrap(t *testing.T) {
	cause := fmt.Errorf("permission denied")
	err := &domain.FileReadError{FilePath: "/etc/secret.yaml", Cause: cause}
	msg := err.Error()
	if !strings.Contains(msg, "/etc/secret.yaml") {
		t.Errorf("expected file path in error message, got %q", msg)
	}
	if !errors.Is(err, cause) {
		t.Error("expected Unwrap to return cause")
	}
}

func TestTemplateRenderError_ErrorAndUnwrap(t *testing.T) {
	cause := fmt.Errorf("undefined variable")
	err := &domain.TemplateRenderError{FilePath: "tmpl.yaml", Detail: "parse", Cause: cause}
	msg := err.Error()
	if !strings.Contains(msg, "tmpl.yaml") {
		t.Errorf("expected file path in error message, got %q", msg)
	}
	if !errors.Is(err, cause) {
		t.Error("expected Unwrap to return cause")
	}
}
