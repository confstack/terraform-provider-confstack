<div align="center">

# ConfStack - Terraform/OpenTofu Provider 

**Merge ordered YAML layers into a single Terraform configuration object**

![Terraform](https://img.shields.io/badge/terraform-%235835CC.svg?style=for-the-badge&logo=terraform&logoColor=white)

</div>

---

## What is confstack?

confstack brings a **GitOps approach to Terraform configuration**. Inspired by [Helmfile](https://helmfile.readthedocs.io/en/latest/)'s layered values, it lets you describe infrastructure parameters as an ordered stack of YAML files — a base layer, an environment layer, a tenant layer, whatever fits your setup — then merges them into a single Terraform object that your HCL can reference.

Your YAML files live alongside your Terraform code (or in a dedicated config repo), get reviewed and merged through pull requests, and drive infrastructure changes through your CI/CD pipeline. Environment promotions become a YAML file addition. Configuration differences between tenants no longer require HCL conditionals. Every change has a Git history.

The provider handles the merging mechanics: recursive deep merge, Go template injection for variables and secrets (`var`/`secret`), and template inheritance for DRY patterns (`_templates`/`_inherit`).

---

## Quick Start

**Step 1 — Install the provider**

```hcl
terraform {
  required_providers {
    confstack = {
      source  = "confstack/confstack"
      version = "~> 1.0"
    }
  }
}

provider "confstack" {}
```

**Step 2 — Write your YAML layers**

```yaml
# config/base.yaml
tags:
  managed_by: opentofu
  team: platform

eks:
  node_size: t3.medium
  min_nodes: 2
```

```yaml
# config/prod.yaml
tags:
  environment: prod

eks:
  node_size: m5.xlarge   # overrides base
  min_nodes: 3
```

**Step 3 — Declare the data source**

```hcl
data "confstack_layered_config" "app" {
  layers = [
    "${path.module}/config/base.yaml",
    "${path.module}/config/${var.environment}.yaml",
  ]
  on_missing_layer = "skip"   # silently ignore absent layer files
}
```

**Step 4 — Use the output**

```hcl
# Nested access
output "node_size" {
  value = data.confstack_layered_config.app.config.eks.node_size
  # → "m5.xlarge" in prod
}

# Flat access (all values as strings)
resource "aws_eks_node_group" "main" {
  instance_types = [data.confstack_layered_config.app.flat_config["eks.node_size"]]
}
```

---

## Data Source: `confstack_layered_config`

### Inputs

| Attribute | Type | Required | Default | Description |
|---|---|---|---|---|
| `layers` | `list(string)` | yes | — | Ordered YAML file paths or glob patterns (`*`, `**`, `?`, `[…]`). Index 0 is lowest priority; last is highest (last wins). Globs expand alphabetically at their position. |
| `on_missing_layer` | `string` | no | `"error"` | How to handle a missing file. One of `"error"`, `"warn"`, `"skip"`. |
| `variables` | `map(string)` | no | `{}` | Values injected via `{{ var "KEY" }}` in YAML templates. Checks this map first, then OS environment. |
| `secrets` | `map(string)` | no | `{}` | Sensitive values injected via `{{ secret "KEY" }}`. Marked sensitive in Terraform. Checks this map first, then OS environment. |
| `flat_separator` | `string` | no | `"."` | Separator used when flattening nested keys into `flat_config`. |

### Outputs

| Attribute | Type | Sensitive | Description |
|---|---|---|---|
| `config` | `dynamic` | no | Fully resolved config. Secrets are redacted as `"(sensitive)"`. |
| `sensitive_config` | `dynamic` | yes | Same as `config` but with secrets in plaintext. |
| `flat_config` | `map(string)` | no | `config` flattened to `flat_separator`-delimited keys. All values converted to strings. |
| `loaded_layers` | `list(string)` | no | Paths of layer files that were successfully loaded. |
| `secret_paths` | `list(string)` | no | Dot-delimited paths in `config` that contain secret values. |

---

## Layer Merge Behavior

Layers are merged in index order — **index 0 is the base (lowest priority), the last entry wins**. Within each merge:

| Value type | Merge strategy |
|---|---|
| Map | Recursively merged; new keys are added, existing keys are overridden |
| Scalar (string, number, bool) | Replaced entirely by the higher-priority value |
| List | Replaced entirely (not concatenated) |
| `null` | Deletes the key from the result (applied predictably at every layer). |

```yaml
# base.yaml
tags:
  managed_by: opentofu
  environment: dev
items: [a, b]

# prod.yaml
tags:
  environment: prod   # overrides; managed_by is kept
items: [c]            # replaces the whole list
to_delete: ~          # null deletes the key
```

Result after `layers = [base.yaml, prod.yaml]`:

```yaml
tags:
  managed_by: opentofu
  environment: prod
items: [c]
# to_delete is gone
```

---

## Templating

YAML files are processed as [Go templates](https://pkg.go.dev/text/template) before YAML parsing. The full [Sprig](http://masterminds.github.io/sprig/) function library is available.

### `var "KEY"` — inject a variable

Looks up `KEY` in `variables`, then falls back to the OS environment. Errors if the key is missing in both. Outputs a JSON-encoded string (safe for YAML contexts).

```yaml
network:
  vpc_id: {{ var "VPC_ID" }}
```

### `secret "KEY"` — inject a secret

Same lookup order as `var`, but the value becomes `"(sensitive)"` in `config` and the real value in `sensitive_config`. The path is recorded in `secret_paths`.

```yaml
database:
  password: {{ secret "DB_PASSWORD" }}
```

### Sprig functions

All Sprig functions are available, including `upper`, `lower`, `default`, `trimAll`, `toJson`, and many more. For example:

```yaml
name: {{ var "SERVICE_NAME" | lower | replace "_" "-" }}
```

> **Note:** Sprig's `env "KEY"` function also reads OS environment variables directly, but it returns an empty string if the key is missing (no error) and does not consult the `variables` map. Prefer `var "KEY"` for consistent error handling.

---

## Inheritance (`_templates` / `_inherit`)

Define reusable config blocks with `_templates` and apply them with `_inherit`. Templates are collected globally from all loaded layers.

```yaml
sqs_queues:
  _templates:
    standard:
      retention: 86400
      dlq: true
      visibility_timeout: 30
    critical:
      retention: 604800
      dlq: true
      visibility_timeout: 30
      dlq_max_retries: 5

  # Single template
  notifications:
    _inherit: standard
    retention: 3600        # entry-level override applied after inheriting

  # Multiple templates + key exclusion
  orders:
    _inherit:
      - template: standard
        except: [dlq]      # skip dlq from standard
      - template: critical # then merge all of critical on top
```

Rules:

- `_inherit` accepts a string (single template name), a list of strings, or a list of `{template, except}` objects.
- Templates are merged left-to-right; the entry's own keys always win last.
- Template names must be globally unique across all loaded layers.
- Templates cannot themselves contain `_inherit`.
- `_templates` and `_inherit` are stripped from all outputs.

---

## Flat Output

`flat_config` collapses all nested keys into a `map(string)` using `flat_separator` (default `.`). All values are converted to their string representation. This is useful for passing individual values directly to resource arguments.

```hcl
data "confstack_layered_config" "app" {
  layers         = ["${path.module}/config/base.yaml"]
  flat_separator = "/"   # optional; use "/" for URL-safe keys
}

resource "aws_db_instance" "main" {
  address = data.confstack_layered_config.app.flat_config["database/host"]
  port    = data.confstack_layered_config.app.flat_config["database/port"]
}
```

---

## Glob Patterns in `layers`

Each entry in `layers` may be a literal path **or** a glob pattern. Glob patterns are expanded to alphabetically sorted concrete file paths at their position — preserving the last-wins merge order.

```hcl
data "confstack_layered_config" "app" {
  layers = [
    "${path.module}/config/base.yaml",
    "${path.module}/config/overrides/*.yaml",   # expands: 01-net.yaml, 02-compute.yaml, …
    "${path.module}/config/secrets/**/*.yaml",  # ** = recursive
  ]
}
```

- `loaded_layers` shows the concrete paths after expansion — useful for debugging.
- A glob that matches zero files respects `on_missing_layer` (error/warn/skip).
- Directories are never matched, only files.

---

## Missing Layer Handling

| `on_missing_layer` | Behavior when a layer file is absent |
|---|---|
| `"error"` (default) | Error — plan/apply fails immediately. |
| `"warn"` | Warning logged; layer is skipped. Resolution continues. |
| `"skip"` | Layer silently skipped. Resolution continues. |

Combine `on_missing_layer = "skip"` with `compact()` to build dynamic, optional layer stacks:

```hcl
data "confstack_layered_config" "app" {
  layers = compact([
    "${path.module}/config/base.yaml",
    "${path.module}/config/${var.environment}.yaml",
    var.tenant != "" ? "${path.module}/config/tenants/${var.tenant}.yaml" : "",
  ])
  on_missing_layer = "skip"
}
```

---

## Development & Versioning

This provider uses [Conventional Commits](https://www.conventionalcommits.org/) for automated semantic versioning and changelog generation.

| Prefix | Type | Resulting Version Change |
|---|---|---|
| `feat:` | Feature | Minor (e.g., 1.0.0 → 1.1.0) |
| `fix:` | Bug Fix | Patch (e.g., 1.0.0 → 1.0.1) |
| `perf:`, `refactor:`, `chore:` | Internal | Patch (if it affects built files) or no release |
| `BREAKING CHANGE:` | Breaking | Major (e.g., 1.0.0 → 2.0.0) |

Commits to the `main` branch will automatically trigger a new version tag and GitHub release if a version-worthy change is detected.

---

## Examples

| Example | Description |
|---|---|
| [`basic/`](examples/data-sources/confstack_layered_config/basic/) | Two-layer merge (base + environment override) |
| [`multi-environment/`](examples/data-sources/confstack_layered_config/multi-environment/) | Dynamic environment layer with `on_missing_layer = "skip"` |
| [`inheritance/`](examples/data-sources/confstack_layered_config/inheritance/) | `_templates` / `_inherit` for DRY configuration |
| [`templating/`](examples/data-sources/confstack_layered_config/templating/) | `var()` / `secret()` template injection |
| [`flat-output/`](examples/data-sources/confstack_layered_config/flat-output/) | `flat_config` for ergonomic resource attribute access |
| [`complete/`](examples/data-sources/confstack_layered_config/complete/) | Full multi-layer stack with env, tenant, `compact()`, secrets |
| [`glob-layers/`](examples/data-sources/confstack_layered_config/glob-layers/) | Glob patterns (`*.yaml`, `**/*.yaml`) in `layers` |
