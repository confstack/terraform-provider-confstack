package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Discoverer implements port/output.FileDiscoverer using the local filesystem.
type Discoverer struct{}

// NewDiscoverer returns a new filesystem-backed Discoverer.
func NewDiscoverer() *Discoverer {
	return &Discoverer{}
}

// Discover scans config_dir and returns DiscoveredFile entries in priority order.
func (d *Discoverer) Discover(ctx context.Context, req domain.ResolveRequest) ([]domain.DiscoveredFile, error) {
	if _, err := os.Stat(req.ConfigDir); os.IsNotExist(err) {
		return nil, &domain.ConfigDirNotFoundError{ConfigDir: req.ConfigDir}
	}

	var files []domain.DiscoveredFile

	// Scan global scope (priority 1-4) then env scope (priority 5-8)
	scopes := []struct {
		dir      string
		scopeName string
		basePriority int
	}{
		{filepath.Join(req.ConfigDir, req.GlobalDir), req.GlobalDir, 1},
		{filepath.Join(req.ConfigDir, req.Environment), req.Environment, 5},
	}

	for _, scope := range scopes {
		entries, err := d.scanDir(ctx, scope.dir, scope.scopeName, scope.basePriority, req)
		if err != nil {
			return nil, err
		}
		files = append(files, entries...)
	}

	// Sort by priority, then lexicographically by filename within the same priority
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Priority != files[j].Priority {
			return files[i].Priority < files[j].Priority
		}
		return filepath.Base(files[i].Path) < filepath.Base(files[j].Path)
	})

	tflog.Debug(ctx, "discovered files", map[string]any{"count": len(files)})
	return files, nil
}

// scanDir reads a scope directory and returns matching DiscoveredFile entries.
// Missing directory is tolerated (returns empty slice).
func (d *Discoverer) scanDir(ctx context.Context, dir string, scopeName string, basePriority int, req domain.ResolveRequest) ([]domain.DiscoveredFile, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		tflog.Debug(ctx, "scope directory does not exist, skipping", map[string]any{"dir": dir})
		return nil, nil
	}
	if err != nil {
		return nil, &domain.FileReadError{FilePath: dir, Cause: err}
	}

	// Case collision detection: normalize all filenames to lowercase and check for duplicates
	lowerToActual := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if existing, ok := lowerToActual[lower]; ok {
			return nil, &domain.CaseCollisionError{
				Dir:   dir,
				FileA: existing,
				FileB: e.Name(),
			}
		}
		lowerToActual[lower] = e.Name()
	}

	ext := "." + req.FileExtension
	var files []domain.DiscoveredFile

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if !strings.HasSuffix(name, ext) {
			continue
		}

		// Parse <prefix>.<slug>.<ext>
		withoutExt := strings.TrimSuffix(name, ext)
		parts := strings.SplitN(withoutExt, ".", 2)
		if len(parts) != 2 {
			// Doesn't match <prefix>.<slug> pattern; skip
			continue
		}
		prefix := parts[0]
		slug := parts[1]

		// Only load files targeting common or the specified tenant
		if slug != req.CommonSlug && (req.Tenant == "" || slug != req.Tenant) {
			tflog.Debug(ctx, "skipping file: slug does not match tenant or common",
				map[string]any{"file": name, "slug": slug, "tenant": req.Tenant})
			continue
		}

		absPath := filepath.Join(dir, name)
		relPath, _ := filepath.Rel(req.ConfigDir, absPath)

		priority := d.computePriority(basePriority, prefix, slug, req)

		files = append(files, domain.DiscoveredFile{
			Path:    absPath,
			RelPath: relPath,
			Scope:   scopeName,
			Prefix:  prefix,
			Slug:    slug,
			Priority: priority,
		})

		tflog.Debug(ctx, "discovered file",
			map[string]any{"path": relPath, "priority": priority})
	}

	return files, nil
}

// computePriority assigns the 1-8 priority level based on scope, prefix, and slug.
//
// Priority table:
//  1. _global / defaults / common
//  2. _global / defaults / tenant
//  3. _global / domain   / common
//  4. _global / domain   / tenant
//  5. env     / defaults / common
//  6. env     / defaults / tenant
//  7. env     / domain   / common
//  8. env     / domain   / tenant
func (d *Discoverer) computePriority(basePriority int, prefix, slug string, req domain.ResolveRequest) int {
	isDefaults := prefix == req.DefaultsPrefix
	isTenant := slug == req.Tenant && slug != req.CommonSlug

	// basePriority is 1 for global, 5 for env
	// Within each scope: defaults=+0, domain=+2; common=+0, tenant=+1
	offset := 0
	if !isDefaults {
		offset += 2
	}
	if isTenant {
		offset += 1
	}
	return basePriority + offset
}
