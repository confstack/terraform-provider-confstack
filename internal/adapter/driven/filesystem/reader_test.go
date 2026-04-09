package filesystem_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/filesystem"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
)

func TestReader_Read_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	want := []byte("key: value\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	r := filesystem.NewReader()
	got, err := r.Read(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestReader_Read_MissingFile(t *testing.T) {
	r := filesystem.NewReader()
	_, err := r.Read(context.Background(), "/nonexistent/path/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	var fre *domain.FileReadError
	if !errors.As(err, &fre) {
		t.Errorf("expected FileReadError, got %T: %v", err, err)
	}
	if fre.FilePath != "/nonexistent/path/file.yaml" {
		t.Errorf("expected FilePath to be set, got %q", fre.FilePath)
	}
}
