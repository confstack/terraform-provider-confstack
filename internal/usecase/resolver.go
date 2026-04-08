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
	reader     portout.FileReader
	parser     portout.YAMLParser
	tmplEngine portout.TemplateEngine
	logger     portout.Logger
	expander   portout.PathExpander
}

// NewResolver creates a new Resolver with all required output adapters injected.
func NewResolver(
	reader portout.FileReader,
	parser portout.YAMLParser,
	tmplEngine portout.TemplateEngine,
	logger portout.Logger,
	expander portout.PathExpander,
) *Resolver {
	return &Resolver{
		reader:     reader,
		parser:     parser,
		tmplEngine: tmplEngine,
		logger:     logger,
		expander:   expander,
	}
}

// Resolve runs the full configuration resolution pipeline.
func (r *Resolver) Resolve(ctx context.Context, req domain.ResolveRequest) (*domain.ResolveResult, error) {
	r.logger.Debug(ctx, "starting config resolution", map[string]any{
		"layers": len(req.Layers),
	})

	// Use a single nonce per Resolve call to prevent cross-run sentinel collisions.
	nonce := uuid.New().String()

	// Step 0: Expand glob patterns in req.Layers to concrete file paths.
	var expandedLayers []string
	for _, entry := range req.Layers {
		if !domain.IsGlobPattern(entry) {
			expandedLayers = append(expandedLayers, entry)
			continue
		}
		matches, err := r.expander.Expand(ctx, entry)
		if err != nil {
			return nil, fmt.Errorf("expanding glob %q: %w", entry, err)
		}
		if len(matches) == 0 {
			switch req.OnMissingLayer {
			case "error":
				return nil, &domain.NoGlobMatchError{Pattern: entry}
			case "warn":
				r.logger.Debug(ctx, "glob matched no files, skipping (on_missing_layer=warn)",
					map[string]any{"pattern": entry})
			case "skip":
				r.logger.Debug(ctx, "glob matched no files, skipping (on_missing_layer=skip)",
					map[string]any{"pattern": entry})
			default:
				return nil, fmt.Errorf("unexpected on_missing_layer value %q", req.OnMissingLayer)
			}
			continue
		}
		expandedLayers = append(expandedLayers, matches...)
	}

	// Step 1: Load and process each layer in order (index 0 = lowest priority).
	var files []domain.DiscoveredFile
	fileData := make(map[string][]map[string]any)
	allSentinels := make(map[string]string)
	var loadedLayers []string

	for i, layerPath := range expandedLayers {
		// Check existence via expander (single-file "glob" returns [] if missing).
		matches, err := r.expander.Expand(ctx, layerPath)
		if err != nil {
			return nil, fmt.Errorf("checking layer %q: %w", layerPath, err)
		}
		if len(matches) == 0 {
			switch req.OnMissingLayer {
			case "error":
				return nil, &domain.LayerNotFoundError{LayerPath: layerPath}
			case "warn":
				r.logger.Debug(ctx, "layer not found, skipping (on_missing_layer=warn)",
					map[string]any{"layer": layerPath})
				continue
			case "skip":
				r.logger.Debug(ctx, "layer not found, skipping (on_missing_layer=skip)",
					map[string]any{"layer": layerPath})
				continue
			default:
				// NewResolveRequest validates this; should be unreachable.
				return nil, fmt.Errorf("unexpected on_missing_layer value %q", req.OnMissingLayer)
			}
		}

		raw, err := r.reader.Read(ctx, layerPath)
		if err != nil {
			return nil, fmt.Errorf("reading layer %q: %w", layerPath, err)
		}

		processed, fileSentinels, err := r.tmplEngine.Process(ctx, raw, layerPath, req, nonce)
		if err != nil {
			return nil, fmt.Errorf("templating layer %q: %w", layerPath, err)
		}
		for k, v := range fileSentinels {
			allSentinels[k] = v
		}

		docs, err := r.parser.ParseMultiDoc(ctx, processed, layerPath)
		if err != nil {
			return nil, fmt.Errorf("parsing layer %q: %w", layerPath, err)
		}

		file := domain.DiscoveredFile{Path: layerPath, Priority: i}
		files = append(files, file)
		fileData[layerPath] = docs
		loadedLayers = append(loadedLayers, layerPath)

		r.logger.Debug(ctx, "loaded layer", map[string]any{
			"layer":    layerPath,
			"priority": i,
			"docs":     len(docs),
		})
	}

	r.logger.Debug(ctx, "layers loaded", map[string]any{"count": len(loadedLayers)})

	// Step 2: Deep merge all files in priority order.
	merged, err := domain.MergeAll(files, fileData)
	if err != nil {
		return nil, fmt.Errorf("deep merge: %w", err)
	}

	r.logger.Trace(ctx, "post-merge tree", map[string]any{"keys": len(merged)})

	// Step 3: Resolve inheritance (_templates / _inherit).
	inherited, err := domain.ResolveInheritance(merged, req.TemplatesKey, req.InheritKey)
	if err != nil {
		return nil, fmt.Errorf("inheritance resolution: %w", err)
	}

	r.logger.Trace(ctx, "post-inheritance tree", map[string]any{"keys": len(inherited)})

	// Step 4: Strip reserved keys (_templates, _inherit).
	cleaned := domain.StripReservedKeys(inherited, req.TemplatesKey, req.InheritKey)

	// Step 5: Resolve secrets (sentinels → real values or redacted values).
	redacted, full, secretPaths, err := domain.ResolveSecrets(cleaned, allSentinels)
	if err != nil {
		return nil, fmt.Errorf("secret resolution: %w", err)
	}

	// Step 6: Flatten redacted output to dot-delimited keys.
	flatOutput := domain.Flatten(redacted, req.FlatSeparator)

	return &domain.ResolveResult{
		Output:          redacted,
		SensitiveOutput: full,
		FlatOutput:      flatOutput,
		LoadedLayers:    loadedLayers,
		SecretPaths:     secretPaths,
	}, nil
}
