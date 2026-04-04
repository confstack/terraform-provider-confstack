package usecase

import (
	"context"
	"fmt"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	portout "github.com/confstack/terraform-provider-confstack/internal/port/output"
	"github.com/google/uuid"
)

// Resolver implements port/input.ConfigResolver. It orchestrates the full resolution pipeline.
type Resolver struct {
	discoverer portout.FileDiscoverer
	reader     portout.FileReader
	parser     portout.YAMLParser
	tmplEngine portout.TemplateEngine
	logger     portout.Logger
}

// NewResolver creates a new Resolver with all required output adapters injected.
func NewResolver(
	discoverer portout.FileDiscoverer,
	reader portout.FileReader,
	parser portout.YAMLParser,
	tmplEngine portout.TemplateEngine,
	logger portout.Logger,
) *Resolver {
	return &Resolver{
		discoverer: discoverer,
		reader:     reader,
		parser:     parser,
		tmplEngine: tmplEngine,
		logger:     logger,
	}
}

// processFiles reads, templates, and parses each discovered file.
// Returns per-file parsed docs, accumulated sentinels, and ordered file paths.
func (r *Resolver) processFiles(ctx context.Context, files []domain.DiscoveredFile, req domain.ResolveRequest, nonce string) (
	fileData map[string][]map[string]any, sentinels map[string]string, loadedFiles []string, err error,
) {
	sentinels = make(map[string]string)
	fileData = make(map[string][]map[string]any, len(files))
	loadedFiles = make([]string, 0, len(files))

	for _, file := range files {
		raw, err := r.reader.Read(ctx, file.Path, req.ConfigDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("reading file %q: %w", file.RelPath, err)
		}

		processed, fileSentinels, err := r.tmplEngine.Process(ctx, raw, file.Path, req, nonce)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("templating file %q: %w", file.RelPath, err)
		}
		for k, v := range fileSentinels {
			sentinels[k] = v
		}

		docs, err := r.parser.ParseMultiDoc(ctx, processed, file.Path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parsing file %q: %w", file.RelPath, err)
		}

		fileData[file.Path] = docs
		loadedFiles = append(loadedFiles, file.RelPath)

		r.logger.Debug(ctx, "loaded file",
			map[string]any{"file": file.RelPath, "priority": file.Priority, "docs": len(docs)})
	}
	return fileData, sentinels, loadedFiles, nil
}

// Resolve runs the full configuration resolution pipeline.
func (r *Resolver) Resolve(ctx context.Context, req domain.ResolveRequest) (*domain.ResolveResult, error) {
	r.logger.Debug(ctx, "starting config resolution",
		map[string]any{
			"environment": req.Environment,
			"tenant":      req.Tenant,
			"config_dir":  req.ConfigDir,
		})

	// Step 2: Discover files
	files, err := r.discoverer.Discover(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("file discovery: %w", err)
	}

	r.logger.Debug(ctx, "discovered files", map[string]any{"count": len(files)})

	// Step 3 + 4: Template + parse YAML for each file
	// Use a single nonce per Resolve call to prevent cross-run sentinel collisions
	nonce := uuid.New().String()
	fileData, allSentinels, loadedFiles, err := r.processFiles(ctx, files, req, nonce)
	if err != nil {
		return nil, err
	}

	// Step 5 + 6: Deep merge all files in priority order
	merged, err := domain.MergeAll(files, fileData)
	if err != nil {
		return nil, fmt.Errorf("deep merge: %w", err)
	}

	r.logger.Trace(ctx, "post-merge tree", map[string]any{"keys": len(merged)})

	// Step 7: Resolve inheritance
	inherited, err := domain.ResolveInheritance(merged, req.TemplatesKey, req.InheritKey)
	if err != nil {
		return nil, fmt.Errorf("inheritance resolution: %w", err)
	}

	r.logger.Trace(ctx, "post-inheritance tree", map[string]any{"keys": len(inherited)})

	// Step 8: Strip reserved keys
	cleaned := domain.StripReservedKeys(inherited, req.TemplatesKey, req.InheritKey)

	// Step 9: Resolve secrets (sentinels → real values / redacted values)
	redacted, full, secretPaths, err := domain.ResolveSecrets(cleaned, allSentinels)
	if err != nil {
		return nil, fmt.Errorf("secret resolution: %w", err)
	}

	return &domain.ResolveResult{
		Output:          redacted,
		SensitiveOutput: full,
		LoadedFiles:     loadedFiles,
		SecretPaths:     secretPaths,
	}, nil
}
