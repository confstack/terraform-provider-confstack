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

func setupConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverer_BasicDiscovery(t *testing.T) {
	dir := setupConfigDir(t)
	writeFile(t, filepath.Join(dir, "_global"), "defaults.common.yaml", "x: 1")
	writeFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", "y: 2")
	writeFile(t, filepath.Join(dir, "prod"), "defaults.common.yaml", "z: 3")

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

	files, err := d.Discover(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestDiscoverer_Priority8Levels(t *testing.T) {
	dir := setupConfigDir(t)
	tenant := "acme"

	writeFile(t, filepath.Join(dir, "_global"), "defaults.common.yaml", "")   // p1
	writeFile(t, filepath.Join(dir, "_global"), "defaults.acme.yaml", "")     // p2
	writeFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", "")    // p3
	writeFile(t, filepath.Join(dir, "_global"), "compute.acme.yaml", "")      // p4
	writeFile(t, filepath.Join(dir, "prod"), "defaults.common.yaml", "")      // p5
	writeFile(t, filepath.Join(dir, "prod"), "defaults.acme.yaml", "")        // p6
	writeFile(t, filepath.Join(dir, "prod"), "compute.common.yaml", "")       // p7
	writeFile(t, filepath.Join(dir, "prod"), "compute.acme.yaml", "")         // p8

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"
	req.Tenant = tenant

	files, err := d.Discover(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 8 {
		t.Fatalf("expected 8 files, got %d", len(files))
	}
	for i, f := range files {
		expectedPriority := i + 1
		if f.Priority != expectedPriority {
			t.Errorf("file %d: expected priority %d, got %d (file: %s)",
				i, expectedPriority, f.Priority, f.RelPath)
		}
	}
}

func TestDiscoverer_MissingDirsTolerared(t *testing.T) {
	dir := setupConfigDir(t)
	// No _global or prod directories created

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

	files, err := d.Discover(context.Background(), req)
	if err != nil {
		t.Errorf("expected no error for missing scope dirs, got: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestDiscoverer_CaseCollision(t *testing.T) {
	dir := setupConfigDir(t)
	writeFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", "")
	writeFile(t, filepath.Join(dir, "_global"), "Compute.common.yaml", "")

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

	_, err := d.Discover(context.Background(), req)
	if err == nil {
		t.Fatal("expected case collision error")
	}
	var cce *domain.CaseCollisionError
	if !errors.As(err, &cce) {
		t.Errorf("expected CaseCollisionError, got %T: %v", err, err)
	}
}

func TestDiscoverer_SkipsNonMatchingSlugs(t *testing.T) {
	dir := setupConfigDir(t)
	writeFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", "")
	writeFile(t, filepath.Join(dir, "_global"), "compute.otherTenant.yaml", "")

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"
	req.Tenant = "acme"

	files, err := d.Discover(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Only compute.common.yaml matches (common slug). compute.otherTenant.yaml is skipped.
	if len(files) != 1 {
		t.Errorf("expected 1 file (only common), got %d", len(files))
	}
}

func TestDiscoverer_ConfigDirNotExist(t *testing.T) {
	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = "/nonexistent/path/12345"
	req.Environment = "prod"

	_, err := d.Discover(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent config_dir")
	}
	var cnfe *domain.ConfigDirNotFoundError
	if !errors.As(err, &cnfe) {
		t.Errorf("expected ConfigDirNotFoundError, got %T: %v", err, err)
	}
}

func TestDiscoverer_LexicographicOrder(t *testing.T) {
	dir := setupConfigDir(t)
	// Multiple domain files at the same priority level (p3: global/domain/common)
	writeFile(t, filepath.Join(dir, "_global"), "storage.common.yaml", "")
	writeFile(t, filepath.Join(dir, "_global"), "networking.common.yaml", "")
	writeFile(t, filepath.Join(dir, "_global"), "compute.common.yaml", "")

	d := filesystem.NewDiscoverer()
	req := domain.DefaultResolveRequest()
	req.ConfigDir = dir
	req.Environment = "prod"

	files, err := d.Discover(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	names := []string{
		filepath.Base(files[0].Path),
		filepath.Base(files[1].Path),
		filepath.Base(files[2].Path),
	}
	expected := []string{"compute.common.yaml", "networking.common.yaml", "storage.common.yaml"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("file %d: expected %s, got %s", i, expected[i], name)
		}
	}
}
