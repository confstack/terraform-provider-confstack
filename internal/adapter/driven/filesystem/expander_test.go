package filesystem_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/filesystem"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExpander_Expand_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), "a: 1\n")
	writeFile(t, filepath.Join(dir, "b.yaml"), "b: 2\n")
	writeFile(t, filepath.Join(dir, "c.yaml"), "c: 3\n")

	e := filesystem.NewExpander()
	matches, err := e.Expand(context.Background(), filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d: %v", len(matches), matches)
	}
	// Verify alphabetical order
	for i := 1; i < len(matches); i++ {
		if matches[i] < matches[i-1] {
			t.Errorf("matches not sorted: %v", matches)
		}
	}
}

func TestExpander_Expand_NoMatches(t *testing.T) {
	dir := t.TempDir()

	e := filesystem.NewExpander()
	matches, err := e.Expand(context.Background(), filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d: %v", len(matches), matches)
	}
}

func TestExpander_Expand_LiteralPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, path, "key: val\n")

	e := filesystem.NewExpander()
	matches, err := e.Expand(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0] != path {
		t.Errorf("expected exactly [%s], got %v", path, matches)
	}
}

func TestExpander_Expand_DoublestarRecursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a", "1.yaml"), "level: a\n")
	writeFile(t, filepath.Join(dir, "b", "2.yaml"), "level: b\n")
	writeFile(t, filepath.Join(dir, "b", "deep", "3.yaml"), "level: deep\n")

	e := filesystem.NewExpander()
	matches, err := e.Expand(context.Background(), filepath.Join(dir, "**", "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d: %v", len(matches), matches)
	}
}

func TestExpander_Expand_DirectoriesExcluded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "file.yaml"), "key: val\n")
	// Create a subdirectory named "subdir.yaml" to verify WithFilesOnly() works
	if err := os.MkdirAll(filepath.Join(dir, "subdir.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}

	e := filesystem.NewExpander()
	matches, err := e.Expand(context.Background(), filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("expected only 1 match (file, not dir), got %d: %v", len(matches), matches)
	}
}
