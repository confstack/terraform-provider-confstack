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
usecase/resolver.go          ← resolution pipeline (orchestrator)
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

0. **Expand** — Expand glob patterns in `layers` to concrete file paths (`doublestar`, `**` supported), sorted alphabetically and spliced in at the pattern's position. An entry prefixed `literal:` skips expansion entirely, for filenames containing `*`, `?`, or `[`. A pattern matching nothing is governed by `on_missing_layer`; under `error` it raises `NoGlobMatchError`.

Steps 1–3 run per layer, in one loop:

1. **Load** — Read each layer file; a missing file respects `on_missing_layer: error|warn|skip` (`error` → `LayerNotFoundError`)
2. **Template** — Process each layer as Go template; `{{ var "KEY" }}` from the variables map, falling back to OS env; `{{ secret "KEY" }}` replaced with a SHA256 sentinel (`__CONFSTACK_SECRET_<hex>__`); a UUID nonce is generated per `Resolve()` call so sentinels differ across runs (prevents cross-run collisions); Sprig functions available
3. **Parse** — Multi-document YAML (supports `---` separators); every document must be a map at the top level

Then, over the accumulated layers:

4. **Merge** — Recursive deep merge, last layer wins; maps merged recursively, lists/scalars replaced; `null` deletes a key; type mismatch → `MergeConflictError`
5. **Inherit** — Resolve `_templates`/`_inherit` directives via bubble-up scope lookup
6. **Strip** — `domain.StripReservedKeys` removes `_templates`/`_inherit` at every depth
7. **Secrets** — Sentinels → `"(sensitive)"` in `Output`, real values in `SensitiveOutput`; both carry `SecretPaths`
8. **Flatten** — `domain.Flatten` collapses `Output` to separator-delimited keys

## Key Domain Types

```go
// domain/config.go
type ResolveRequest struct {
    Layers         []string          // glob patterns allowed; "literal:" prefix forces an exact path
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
    FlatOutput      map[string]any   // separator-delimited keys; leaf values keep their native Go types
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

## Domain Errors

All errors live in `internal/domain/errors.go`. Key types:

| Error | When |
|---|---|
| `MergeConflictError` | Type mismatch at same key during deep merge (e.g. map vs scalar) |
| `TemplateNotFoundError` | `_inherit` references a template name that doesn't exist |
| `DuplicateTemplateError` | Same template name defined in multiple `_templates` blocks |
| `TemplateWithInheritError` | A template definition itself contains `_inherit` (forbidden) |
| `MissingVariableError` | `var()` or `secret()` key not in variables map and not in OS env |
| `TemplateRenderError` | Go template parsing or execution failure |
| `LayerNotFoundError` | Layer file missing and `on_missing_layer = "error"` |
| `NoGlobMatchError` | Glob pattern matched no files and `on_missing_layer = "error"` |
| `ParseError` | Invalid YAML, non-map top-level document, or a non-string key in a nested map |
| `FileReadError` | `os.ReadFile` failed on a layer that exists |

## Spec-Driven Workflow (OpenSpec)

Behavior is specified in `openspec/specs/<capability>/spec.md` before it is built. Seven capabilities
are documented: `layer-resolution`, `layer-templating`, `yaml-parsing`, `config-merge`,
`template-inheritance`, `secret-redaction`, `terraform-data-source`.

The loop is `/opsx:propose <idea>` → review the generated `proposal.md` / `design.md` / `tasks.md` →
`/opsx:apply` → `/opsx:archive`. Archiving merges the change's delta specs back into
`openspec/specs/`, so the specs stay current without a separate documentation pass.

- A change proposes **deltas** (`## ADDED` / `## MODIFIED` / `## REMOVED Requirements`), never a
  rewritten spec. A `MODIFIED` block must carry the full requirement including every scenario that
  survives — `openspec validate` and `openspec archive` reject one that silently drops a scenario.
- Scenarios are written as `GIVEN` / `WHEN` / `THEN` bullets. Requirements state observable behavior
  with `SHALL`; implementation detail belongs in `design.md`.
- `openspec/config.yaml` carries the project context and the per-artifact rules injected into every
  proposal. Update it when conventions change.
- Useful checks: `openspec validate --specs`, `openspec list --specs`, `openspec show <name>`.

When behavior changes, update the affected spec in the same change — a spec that drifts from the code
is worse than no spec.

## Conventions

- Conventional Commits enforced on PRs (`feat:`, `fix:`, `BREAKING CHANGE:`)
- Semver automated from commit history via `release-please`; docs in `docs/` are generated (`go generate ./...`) and must be checked in
- Go 1.25.0+ required (pinned in `go.mod`)
- Errors defined in `internal/domain/errors.go` — add new error types there, not inline

## Pull Requests

**Write the body from `.github/PULL_REQUEST_TEMPLATE.md`.** Read it first and keep its five
sections, in order and under their own headings: `Description`, `Discussion`,
`Overview of changes`, `External requirements`, `Impact`. Do not invent a different structure —
`gh pr create --body-file` bypasses the template silently, so it has to be applied by hand.

Section conventions, as established by the merged PRs (#6, #17):

- **Description** — what changed and why, in prose. The diff already shows the how.
- **Discussion** — trade-offs, alternatives rejected, deliberate deviations from a documented
  default, and open questions for the reviewer to settle. This is where a judgment call gets
  recorded; leave it blank only if there genuinely was none.
- **Overview of changes** — a `| Area | Change |` table mapping each touched path or module to
  what changed in it, so a reviewer can navigate the diff without archaeology. Describe the diff,
  not the domain.
- **External requirements** — anything outside this repo needed to make the change work: new
  tooling and its version floor, env vars, provider config, registry permissions, CI secrets,
  dependency upgrades. Write `None.` when there are none, and say so explicitly when `go.mod`
  is untouched.
- **Impact** — who is affected and how: breaking changes, behavior differences for existing
  users, docs needing updates, whether release-please will cut a release.

More generally: before writing any artifact the repo might already have a convention for — a PR
body, an issue, a changelog entry, a config file — look for an existing template or a recent
example and follow it, rather than composing a structure from scratch.
