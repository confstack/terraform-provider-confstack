# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build      # Build provider binary to bin/
make test       # Unit tests (internal/...)
make bdd        # BDD tests (Ginkgo/Gomega)
make e2e        # Acceptance tests (requires TF_ACC=1)
make cover      # Coverage report
make lint       # golangci-lint
make fmt        # gofmt + goimports
make install    # Build + install to local TF plugin dir
make generate   # go generate
```

**Single test targeting:**
```bash
# Unit test
go test ./internal/domain/... -v -run TestDeepMerge

# BDD spec
go test ./tests/bdd/... -v -run "Merge Priority"

# E2E acceptance test
TF_ACC=1 go test ./tests/e2e/... -v -run TestAccLayeredConfigDataSource_basic
```

## Architecture

This is a **hexagonal (clean) architecture** provider. Dependencies always point inward — the domain never imports adapters.

```
adapter/driving/terraform/   ← Terraform framework wiring (provider, data source, mapper)
    ↓ calls
port/input/                  ← ConfigResolver interface (input port)
    ↓ implemented by
usecase/resolver.go          ← 6-step resolution pipeline (orchestrator)
    ↓ calls
domain/                      ← Pure Go business logic (merge, inherit, flatten, secrets, errors)
    ↓ via interfaces
port/output/                 ← FileReader, YAMLParser, TemplateEngine, Logger (output ports)
    ↓ implemented by
adapter/driven/              ← filesystem, yaml, template, logging (driven adapters)
```

The domain (`internal/domain/`) has **zero external dependencies**. All I/O is behind port interfaces injected at construction time in `adapter/driving/terraform/provider.go`.

## Resolution Pipeline

`usecase/resolver.go` runs these steps in order:
1. **Load** — Read each layer file; respects `on_missing_layer: error|warn|skip`
2. **Template** — Process each layer as Go template; `{{ var "KEY" }}` from variables/env, `{{ secret "KEY" }}` replaced with UUID sentinel (Sprig functions available)
3. **Parse** — Multi-document YAML (supports `---` separators)
4. **Merge** — Recursive deep merge, last layer wins; maps merged recursively, lists/scalars replaced; `null` deletes a key; type mismatch → `MergeConflictError`
5. **Inherit** — Resolve `_templates`/`_inherit` directives; templates stripped from final output
6. **Secrets** — Sentinels → `"(sensitive)"` in `config`, real values in `sensitive_config`; all outputs include `secret_paths`

## Key Domain Types

```go
// domain/config.go
type ResolveRequest struct {
    Layers         []string
    OnMissingLayer string            // "error" | "warn" | "skip"
    Variables      map[string]string // {{ var "KEY" }}
    Secrets        map[string]string // {{ secret "KEY" }}
    TemplatesKey   string            // default: "_templates"
    InheritKey     string            // default: "_inherit"
    FlatSeparator  string            // default: "."
}

type ResolveResult struct {
    Output          map[string]any   // secrets redacted as "(sensitive)"
    SensitiveOutput map[string]any   // real secret values
    FlatOutput      map[string]any   // dot-delimited string keys
    LoadedLayers    []string
    SecretPaths     map[string]bool
}
```

## Provider

Single read-only data source: `confstack_layered_config`. No managed resources.

- `adapter/driving/terraform/provider.go` — Provider registration, dependency injection
- `adapter/driving/terraform/data_config.go` — Data source schema + `Read()` implementation
- `adapter/driving/terraform/mapper.go` — `map[string]any` ↔ Terraform Dynamic/attr.Value conversion

## Testing Layout

- `internal/domain/*_test.go` — Unit tests for each domain file
- `tests/bdd/` — Ginkgo/Gomega behavioral specs with fixture YAML in `testdata/`
- `tests/e2e/` — Terraform acceptance tests (require `TF_ACC=1`) with HCL configs in `testdata/`

## Conventions

- Conventional Commits enforced on PRs (`feat:`, `fix:`, `BREAKING CHANGE:`)
- Semver automated from commit history
- Errors defined in `internal/domain/errors.go` — add new error types there, not inline
