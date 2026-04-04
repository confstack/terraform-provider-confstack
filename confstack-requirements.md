# terraform-provider-confstack — Requirements Document

## 1. Purpose

`terraform-provider-confstack` is an OpenTofu/Terraform provider that resolves layered, hierarchical YAML configuration into a single merged output object. It is designed for multi-environment, multi-tenant infrastructure projects that use a flat HCL codebase (one set of modules) with configuration varying by environment and tenant.

It strictly adheres to GitOps principles, reading configuration exclusively from the local filesystem (avoiding external HTTP or S3 fetches) to ensure setups are fully declarative, auditable, and securely self-contained.

The provider replaces the need to manually wire together `yamldecode`, `fileset`, deep merge providers, and custom defaults logic. It encapsulates the entire config resolution pipeline — file discovery, templating, load ordering, deep merging, defaults application, and template inheritance — into a single data source.

---

## 2. Problem Statement

Managing infrastructure configuration across multiple environments (dev, staging, prod) and multiple tenants requires a way to:

- Define shared defaults once and override them selectively per environment or tenant.
- Avoid duplicating configuration across environments and tenants.
- Support nested resource definitions (e.g., a map of SQS queues, each with their own properties) where overrides should merge at the leaf level, not replace entire subtrees.
- Apply default properties to all entries in a resource map without repeating them per entry.
- Support template/inheritance patterns where an entry can inherit properties from a named template.
- Safely inject sensitive values and dynamic Terraform data into the configuration without committing plain-text secrets to version control.

Today, solving this in OpenTofu requires stitching together multiple tools (deep merge providers, fileset loops, manual defaults logic in HCL). HCL cannot express recursive transforms, making generic defaults application and inheritance resolution impossible in pure HCL. This provider solves all of these problems in a single, convention-based data source.

---

## 3. Concepts and Terminology

| Term | Definition |
|------|-----------|
| **Scope** | A directory representing a level in the config hierarchy. There are two scope types: `_global` (applies to all environments) and `<environment>` (applies to a specific environment). |
| **Slug** | The filename segment that determines tenant targeting. A file named `networking.common.yaml` targets all tenants. A file named `networking.acme.yaml` targets only tenant `acme`. |
| **Common file** | A YAML file with the slug `common` (e.g., `config.common.yaml`). Applies to all tenants. |
| **Tenant file** | A YAML file with a tenant name as slug (e.g., `config.acme.yaml`). Applies only to that tenant. |
| **Domain file** | A YAML file whose prefix identifies a logical domain (e.g., `networking`, `compute`, `storage`). Used to split large configs into manageable pieces. |
| **Defaults file** | A YAML file with the prefix `defaults` (e.g., `defaults.common.yaml`). Loaded before domain files within the same scope and slug, providing baseline values. |
| **Templating** | Pre-processing step using Go templates before YAML parsing. Used to inject dynamic data via the `{{ var "KEY" }}` function or sensitive data via `{{ secret "KEY" }}`, which read from the data source's `variables` and `secrets` inputs or fall back to environment variables. |
| **Deep merge** | A recursive merge of nested maps where only leaf values are replaced by higher-priority sources. Lower-priority keys that are not present in higher-priority sources are preserved. |
| **`_templates` key** | A reserved key within a map that defines named templates. Templates are not emitted in the output. They exist only to be referenced via the `_inherit` key. |
| **`_inherit` key** | A reserved key on a map entry that references templates from `_templates`. Supports single inheritance (string) or multiple inheritance (list of objects with optional `except` clauses). The entry deep merges the template's properties as its base, then applies its own overrides. |

---

## 4. Directory Convention

The provider reads configuration from a directory tree with this structure:

```
<config_dir>/
├── _global/                              # Global scope (all environments)
│   ├── defaults.common.yaml             # Global defaults for all tenants
│   ├── defaults.<tenant>.yaml           # Global defaults for a specific tenant
│   ├── <domain>.common.yaml             # Global domain config for all tenants
│   └── <domain>.<tenant>.yaml           # Global domain config for a specific tenant
├── <environment>/                        # Environment scope
│   ├── defaults.common.yaml             # Environment defaults for all tenants
│   ├── defaults.<tenant>.yaml           # Environment defaults for a specific tenant
│   ├── <domain>.common.yaml             # Environment domain config for all tenants
│   └── <domain>.<tenant>.yaml           # Environment domain config for a specific tenant
```

### File naming pattern

Every YAML file follows the pattern: `<prefix>.<slug>.yaml`

- `<prefix>`: Either `defaults` (loaded first) or any domain name (e.g., `networking`, `compute`, `storage`, `config`).
- `<slug>`: Either `common` (applies to all tenants) or a tenant identifier (applies only to that tenant).

### Examples

```
config/
├── _global/
│   ├── defaults.common.yaml          # Base defaults: tags, billing mode, retention
│   ├── defaults.acme.yaml            # Acme-specific global defaults
│   ├── networking.common.yaml        # VPC, subnets shared across all
│   ├── compute.common.yaml           # EKS, instance defaults
│   └── compute.acme.yaml             # Acme compute overrides (larger instances)
├── dev/
│   ├── defaults.common.yaml          # Dev-wide defaults (smaller instances)
│   ├── compute.common.yaml           # Dev compute config
│   └── storage.acme.yaml             # Acme dev storage overrides
└── prod/
    ├── defaults.common.yaml          # Prod-wide defaults (HA, multi-AZ)
    ├── compute.acme.yaml             # Acme prod compute (large instances)
    └── storage.acme.yaml             # Acme prod storage (encryption, replication)
```

---

## 5. Merge Order (Resolution Priority)

Files are loaded and merged in the following order. **Later entries take precedence** (override earlier entries) at every leaf in the deep merge:

```
Priority (lowest → highest):

1. _global / defaults.common.yaml        ← universal baseline
2. _global / defaults.<tenant>.yaml      ← tenant-wide baseline
3. _global / <domain>.common.yaml        ← global domain (all tenants)
4. _global / <domain>.<tenant>.yaml      ← global domain (specific tenant)
5. <env>   / defaults.common.yaml        ← env-wide baseline
6. <env>   / defaults.<tenant>.yaml      ← env + tenant baseline
7. <env>   / <domain>.common.yaml        ← env domain (all tenants)
8. <env>   / <domain>.<tenant>.yaml      ← env domain (specific tenant)
```

### Optional Tenant
If the `tenant` input is omitted or empty, all levels targeting `<tenant>` (levels 2, 4, 6, and 8) are skipped. Only `common` files will be merged.

### Multiple Documents (YAML `---`)
If a single YAML file contains multiple documents (separated by `---`), they are processed sequentially. Each document deep-merges into the result of the previous documents from the same file (last-write-wins).

### Determinism
Within the same priority level (e.g., multiple domain files at level 3), files are merged in **lexicographic order** by filename. This is deterministic but the order between unrelated domain files should not matter in practice because they define different top-level keys.

### Merge rules

- **Maps**: Recursively deep merged. Keys from higher-priority sources override same-path keys from lower-priority sources. Keys not present in higher-priority sources are preserved.
- **Lists**: Replaced entirely (not appended). A list in a higher-priority source replaces the entire list from a lower-priority source.
- **Scalars**: Replaced by higher-priority source.
- **Null values**: A null value in a higher-priority source removes the key from the output (explicit deletion/tombstone), regardless of the previous value's type.

---

## 6. Type Conflicts and Edge Cases

To ensure configuration integrity and prevent accidental data loss, the provider enforces strict type matching during the merge process:

- **Type Mismatch Error**: If a higher-priority source provides a value of a different type than the existing value at the same path (e.g., a scalar replacing a map, or a map replacing a scalar), the provider MUST return a plan-time error.
- **Allowed Transitions**:
    - **Any Type → Null**: Valid (deletes the key).
    - **Null → Any Type**: Valid (re-creates the key).
    - **Map → Map**: Valid (recursive merge).
    - **List → List**: Valid (full replacement).
    - **Scalar → Scalar**: Valid (full replacement).
- **Implicit Map Creation**: If a higher-priority source defines a nested key (e.g., `a.b.c: 1`) where `a` was previously a scalar, this is treated as a type mismatch error for `a`.
- **Duplicate Domains**: If multiple files for the same domain exist at the same priority level, they are merged in lexicographic order. The same strict type rules apply.
- **Error Context**: All type mismatch errors MUST identify the full JSON path of the conflicting key and the types found in both the lower-priority and higher-priority sources.

---

## 7. Templates and Inheritance

After the full deep merge is completed across all 8 levels, the provider resolves the `_templates` and `_inherit` keywords.

### Template Visibility and Scoping
- **Bubble-up Resolution**: When an entry contains an `_inherit` key, the provider looks for the requested template in the sibling `_templates` map. If not found, it recursively searches for a `_templates` map in each parent map up to the root of the configuration tree.
- **Global Uniqueness**: Template names MUST be globally unique across the entire merged configuration tree. Defining the same template name in multiple `_templates` blocks will cause a plan-time error to prevent shadowing and confusion.
- **Timing**: Because inheritance happens *after* the merge, templates defined in `_global/defaults.common.yaml` are visible to entries defined in `<env>/<domain>.<tenant>.yaml`, provided they are within a parent path where the template is visible via bubble-up resolution. This allows for a "Global Library" of templates that can be used at any environment or tenant level.

### `_templates` key

A reserved key within a map that holds named template definitions. Templates are map entries that can be referenced by other entries via `_inherit`. The `_templates` key and all its contents are stripped from the final output.

### `_inherit` key

A reserved key on any map entry that references templates within the nearest `_templates` sibling. 
It can be provided as:
- A **string**: for simple, single inheritance.
- A **list of strings**: for shorthand multiple inheritance (e.g., `_inherit: [base, standard]`).
- A **list of objects**: for multiple inheritance with fine-grained control. Each object must define a `template` string, and can optionally define an `except` list of strings to omit specific inherited keys.

### Behavior

1. Look up the template(s) by name in the `_templates` map at the same level.
2. If `_inherit` is a list (of strings or objects), process the templates sequentially from top to bottom.
3. Deep merge: inherited template values act as the base, subsequent templates overlay them, and the entry's own values act as the final override.
4. If an `except` list is provided for a template (in the object format), those specific keys are excluded from the inherited properties.
5. Remove the `_inherit` key from the entry.
6. Remove the `_templates` key from the parent map.

### Example

**Input:**

```yaml
sqs_queues:
  _templates:
    standard:
      retention: 86400
      dlq: true
      visibility_timeout: 30
    high_retention:
      retention: 604800
      dlq: true
      visibility_timeout: 30
    critical:
      retention: 604800
      dlq: true
      visibility_timeout: 30
      dlq_max_retries: 5

  # Uses simple string inheritance
  notifications:
    _inherit: standard
    retention: 3600

  # Uses multiple inheritance with 'except'
  orders:
    _inherit:
      - template: standard
        except: 
          - dlq
      - template: critical
    visibility_timeout: 120
```

**Resolution of `orders`:**

1. Process first inheritance (`standard` except `dlq`): `{ retention: 86400, visibility_timeout: 30 }`
2. Process second inheritance (`critical`): merges on top. Resolved `critical` = `{ retention: 604800, dlq: true, visibility_timeout: 30, dlq_max_retries: 5 }`.
3. Merged inherited base: `{ retention: 604800, visibility_timeout: 30, dlq: true, dlq_max_retries: 5 }`
4. Entry overrides (`visibility_timeout: 120`) applied on top.

**Output:**

```yaml
sqs_queues:
  notifications:
    retention: 3600
    dlq: true
    visibility_timeout: 30
  orders:
    retention: 604800
    visibility_timeout: 120
    dlq: true
    dlq_max_retries: 5
```

### Edge cases

- Templates MUST NOT contain the `_inherit` key. A `_templates` entry that attempts to inherit from another template must produce an error.
- Referencing a non-existent template must produce an error with the template name and the entry that referenced it.
- `_templates` and `_inherit` must be processed at every nesting depth, not just top-level maps.

---

## 8. Processing Pipeline

The full resolution pipeline is:

```
1. Input Normalization → convert `environment` and `tenant` inputs to lowercase
2. File Discovery      → scan config_dir, match files by naming convention (error on case-only collisions)
3. Templating          → process files as Go templates to inject variables and secrets
4. File Loading        → parse templated output as YAML
5. Ordering            → sort files into the 8-level priority order
6. Deep Merge          → recursively merge all files into a single tree (with strict type checks)
7. Inheritance         → walk tree, resolve `_templates` + `_inherit` via bubble-up lookup
8. Cleanup             → strip all `_templates`, `_inherit` keys from output
9. Output              → return the resolved map
```

---

## 9. Provider Interface

### Provider block

```hcl
terraform {
  required_providers {
    confstack = {
      source  = "<namespace>/confstack"
      version = "~> 1.0"
    }
  }
}

provider "confstack" {}
```

The provider itself requires no configuration. All parameters are on the data source.

### Data source: `confstack_config`

```hcl
data "confstack_config" "this" {
  # Required
  config_dir  = string  # Path to the configuration directory (absolute or relative to the module)
  environment = string  # Environment name (must match a subdirectory name in config_dir)

  # Optional (with defaults)
  tenant         = string      # Tenant identifier. Default: "" (skips tenant-specific files)
  variables      = map(any)    # Standard variables for Go templating. Default: {}
  secrets        = map(any)    # Sensitive variables for Go templating. Default: {}
  global_dir     = string      # Name of the global scope directory. Default: "_global"
  common_slug    = string      # Slug used for files that apply to all tenants. Default: "common"
  defaults_prefix = string     # Filename prefix for defaults files. Default: "defaults"
  templates_key  = string      # Reserved key name for templates. Default: "_templates"
  inherit_key    = string      # Reserved key name for inheritance. Default: "_inherit"
  file_extension = string      # File extension to match. Default: "yaml"
}
```

### Outputs

```hcl
data "confstack_config" "this" { ... }

# The fully resolved configuration map
output "config" {
  value = data.confstack_config.this.output  # type: any (dynamic map)
}

# Metadata about which files were loaded and in what order (for debugging)
output "loaded_files" {
  value = data.confstack_config.this.loaded_files  # type: list(string)
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `output` | `any` | The fully resolved configuration map after deep merge and inheritance. |
| `loaded_files` | `list(string)` | Ordered list of file paths that were discovered and loaded, in merge priority order. Useful for debugging. |

---

## 10. Functional Requirements

### FR-01: Input Normalization and Discovery

- The provider MUST convert `environment` and `tenant` inputs to lowercase before file matching.
- The provider MUST scan `config_dir/<global_dir>/` and `config_dir/<environment>/` for files matching the pattern `<prefix>.<slug>.<file_extension>`.
- The provider MUST only load files where `<slug>` is either `<common_slug>` or `<tenant>`.
- If `tenant` is empty, only `<common_slug>` files are loaded.
- **Case-Insensitivity Collision**: If two files exist in the same directory that differ only by case (e.g., `app.common.yaml` and `App.common.yaml`), the provider MUST return a plan-time error to ensure deterministic behavior.
- Files with slugs that match neither `common_slug` nor the specified `tenant` MUST be ignored.
- The provider MUST NOT recurse into subdirectories within a scope directory.
- If `config_dir` does not exist, the provider MUST return an error.
- If `config_dir/<environment>/` does not exist, the provider MUST treat it as empty (no error).
- If `config_dir/<global_dir>/` does not exist, the provider MUST treat it as empty (no error).

### FR-02: Templating

- The provider MUST process each discovered file as a Go template before parsing it as YAML.
- The templating engine MUST provide a `var` function (e.g., `{{ var "KEY" }}`) and a `secret` function (e.g., `{{ secret "KEY" }}`).
- The `var` and `secret` functions MUST output values formatted as valid YAML/JSON to ensure safe injection without requiring the user to manually quote the template tags.
- **Secret Tracking**: The provider MUST internally track the JSON paths of any values that originate from the `secret` function. The specific tracking mechanism (e.g., injecting magic strings during templating and resolving them post-YAML parsing) is an implementation detail.
- The `var` function MUST first check the `variables` map provided to the data source.
- The `secret` function MUST first check the `secrets` map provided to the data source.
- Both functions MUST fall back to checking the OS environment variables if the key is not found in the respective map.
- The template context MUST include `.Environment` and `.Tenant` variables, allowing files to reference the current scope directly (e.g., `{{ .Environment }}`).
- The templating engine MUST include standard helper functions (Sprig library) for string manipulation, encoding, and math.
- If a key requested via `var` or `secret` is missing from both the map and the environment, the provider MUST return a plan-time error to ensure missing data is explicitly caught.

### FR-03: File Loading

- The provider MUST parse the templated output of each discovered file as YAML.
- **Multiple Documents**: The provider MUST support files containing multiple YAML documents separated by `---`. Documents within a file MUST be merged sequentially in the order they appear.
- If a YAML file contains a syntax error, the provider MUST return an error identifying the file and the parse error.
- An empty YAML file MUST be treated as an empty map `{}`.
- The provider MUST support standard YAML 1.1 or 1.2 features (maps, lists, scalars, anchors, aliases).

### FR-04: Merge Ordering

- The provider MUST merge files in the exact 8-level priority order specified in Section 5.
- Within the same priority level, files MUST be merged in lexicographic order by filename.
- The merge order MUST be deterministic across runs given the same filesystem state.

### FR-05: Deep Merge

- The provider MUST perform recursive deep merge of maps.
- Maps MUST be merged recursively: keys from higher-priority sources override same-path keys; keys absent in higher-priority sources are preserved.
- Lists MUST be replaced entirely (not concatenated).
- Scalar values MUST be replaced by higher-priority sources.
- A `null` value in a higher-priority source MUST remove the key from the output.
- **Strict Type Matching**: Any attempt to replace a map with a non-map (scalar/list) or vice-versa MUST produce an error.

### FR-06: Template and Inheritance Resolution

- After the full deep merge, the provider MUST walk the entire output tree.
- **Bubble-up Resolution**: For any entry with an `inherit_key`, the provider MUST look for the requested template in the sibling `templates_key` map first. If not found, it MUST recursively search parent maps up to the root.
- **Unique Templates**: Template names MUST be globally unique across the entire merged tree. Defining the same template name in multiple locations MUST produce an error.
- The `inherit` value can be a string (single template), a list of strings (shorthand multiple inheritance), or a list of objects (multiple templates with optional `except` arrays).
- When a list is provided (strings or objects), templates MUST be merged sequentially from top to bottom.
- Templates MUST NOT contain the `_inherit` key. A `_templates` entry that attempts to inherit from another template must produce an error.
- References to non-existent templates MUST produce an error identifying the entry and the missing template name.
- All `templates_key` blocks and `inherit_key` entries MUST be stripped from the final output.
- Templates and inheritance MUST be resolved at every depth in the tree, not only at the top level.

### FR-07: Output

- The provider MUST output the fully resolved map as a dynamic type (`any`) so it can represent arbitrary nesting.
- **Granular Sensitivity**: The provider MUST map any paths tracked as secrets (via the `secret` function) to granular `Sensitive: true` flags within the underlying `types.Dynamic` representation of the output. This ensures only specific leaf nodes are masked in the CLI output, leaving the rest of the map visible.
- The output MUST NOT contain any `templates_key` or `inherit_key` entries.
- The provider MUST output a `loaded_files` attribute listing all files that were loaded, in the order they were merged.

### FR-08: Missing Files Tolerance

- Missing scope directories (`_global`, `<environment>`) MUST be tolerated (treated as empty).
- Missing files at any priority level MUST be tolerated (that level is simply skipped).
- The provider MUST work correctly even if only a single file exists in the entire config directory.

### FR-09: Plan-Time Behavior

- The data source MUST be fully resolvable at plan time (no apply-time side effects).
- Changes to any YAML file in the config directory MUST be reflected in the next `tofu plan`.

---

## 11. Non-Functional Requirements

### NFR-01: Performance

- File discovery and loading MUST complete in under 1 second for a config directory with up to 200 YAML files.
- Deep merge and inheritance resolution MUST complete in under 500ms for a merged config tree with up to 10,000 keys.
- The provider SHOULD minimize memory allocations during deep merge.

### NFR-02: Error Messages

- All errors MUST include the file path (when relevant) and a human-readable description of the problem.
- YAML parse errors MUST include the line number if available.
- Template inheritance errors MUST identify the invalid template (e.g., "template 'critical' cannot contain '_inherit'").
- Missing template errors MUST include the entry path and the template name (e.g., "entry 'sqs_queues.orders' references template 'critical' which does not exist in '_templates'").
- Type mismatch errors (e.g., trying to deep merge a map with a scalar) MUST identify the conflicting key path and the two sources.

### NFR-03: Compatibility

- The provider MUST work with OpenTofu >= 1.6 and Terraform >= 1.6.
- The provider MUST be publishable to the Terraform Registry and the OpenTofu Registry.
- The provider MUST follow the Terraform Plugin Framework (not the legacy SDK).
- The provider MUST be written in Go.

### NFR-04: Testing

- Unit tests MUST cover all deep merge scenarios: map+map, map+scalar conflict (error), list replacement, null deletion.
- Unit tests MUST cover inheritance chains of depth 1, 2, and 3+.
- Unit tests MUST cover circular inheritance detection.
- Unit tests MUST cover the full 8-level merge priority order.
- Integration tests (acceptance tests) MUST use the Terraform Plugin Testing framework with real `tofu plan` / `tofu apply` runs.
- Test coverage MUST be >= 90% on the core resolution logic.

### NFR-05: Documentation

- The provider MUST include Terraform Registry-compatible documentation in `docs/`.
- Documentation MUST include a full example with a sample directory structure, sample YAML files, and the corresponding HCL usage.
- Documentation MUST explain the merge priority order with a diagram or table.
- Documentation MUST explain the `_templates` and `_inherit` keywords with before/after examples.
- Documentation MUST explain how to use Go templating and the `var`/`secret` functions for variable injection.

### NFR-06: Logging

- The provider SHOULD support Terraform log levels (TF_LOG).
- At `DEBUG` level, the provider SHOULD log each file as it is loaded and the merge order.
- At `TRACE` level, the provider SHOULD log the intermediate state after each merge step.

### NFR-07: Determinism

- Given identical filesystem state and identical inputs, the provider MUST produce identical output on every run.
- There MUST be no reliance on map iteration order from the Go runtime; all maps MUST be processed in sorted key order where order matters.

### NFR-08: Security

- **State File Storage**: The provider documentation MUST explicitly state that the resolved configuration (including injected secrets) is stored in plaintext in the Terraform/OpenTofu `.tfstate` file.
- The provider MUST NOT make any network calls (no HTTP, S3, etc.), adhering strictly to GitOps local-file configuration.
- The provider MUST only read files within the specified `config_dir`.
- The provider MUST NOT follow symlinks outside of `config_dir`.

---

## 12. Usage Example

### Directory structure

```
infra/
├── main.tf
├── config/
│   ├── _global/
│   │   ├── defaults.common.yaml
│   │   ├── compute.common.yaml
│   │   ├── databases.common.yaml
│   │   └── compute.acme.yaml
│   ├── dev/
│   │   ├── defaults.common.yaml
│   │   └── compute.common.yaml
│   └── prod/
│       ├── defaults.common.yaml
│       ├── compute.acme.yaml
│       └── storage.acme.yaml
```

### Sample YAML files

**config/_global/databases.common.yaml**
```yaml
databases:
  main:
    host: db.example.internal
    # The 'secret' function checks 'secrets' map, then env vars
    password: {{ secret "DB_PASSWORD" }}
    # The 'var' function checks 'variables' map, then env vars
    vpc_id: {{ var "VPC_ID" }}
    engine: postgres
```

**config/_global/defaults.common.yaml**
```yaml
tags:
  managed_by: opentofu
  team: platform

sqs_queues:
  _templates:
    base:
      retention: 86400
      dlq: true
      visibility_timeout: 30

dynamodb_tables:
  _templates:
    base:
      billing_mode: PAY_PER_REQUEST

s3_buckets:
  _templates:
    base:
      versioning: true
      encryption: AES256
```

**config/_global/compute.common.yaml**
```yaml
eks:
  node_size: t3.medium
  min_nodes: 2
  max_nodes: 10

sqs_queues:
  _templates:
    standard:
      retention: 86400
      dlq: true
      visibility_timeout: 30
    high_retention:
      retention: 604800
      dlq: true
      visibility_timeout: 30
    critical:
      retention: 604800
      dlq: true
      visibility_timeout: 30
      dlq_max_retries: 5

  orders:
    _inherit: 
      - template: standard
        except: 
          - dlq
      - template: critical
  notifications:
    _inherit: standard
```

**config/prod/compute.acme.yaml**
```yaml
eks:
  node_size: m5.xlarge
  min_nodes: 3
  max_nodes: 50

sqs_queues:
  orders:
    visibility_timeout: 120
  payments:
    _inherit: critical
    retention: 86400
```

### HCL usage

```hcl
terraform {
  required_providers {
    confstack = {
      source  = "<namespace>/confstack"
      version = "~> 1.0"
    }
  }
}

variable "environment" {
  type = string
}

variable "tenant" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

module "network" {
  source = "./modules/network"
}

data "confstack_config" "this" {
  config_dir  = "${path.module}/config"
  environment = var.environment
  tenant      = var.tenant

  variables = {
    VPC_ID = module.network.vpc_id
  }

  secrets = {
    DB_PASSWORD = var.db_password
  }
}

locals {
  config = data.confstack_config.this.output
}

module "sqs" {
  for_each = lookup(local.config, "sqs_queues", {})
  source   = "./modules/sqs"

  name   = each.key
  config = each.value
  tags   = lookup(local.config, "tags", {})
}

module "dynamodb" {
  for_each = lookup(local.config, "dynamodb_tables", {})
  source   = "./modules/dynamodb"

  name   = each.key
  config = each.value
  tags   = lookup(local.config, "tags", {})
}

module "s3" {
  for_each = lookup(local.config, "s3_buckets", {})
  source   = "./modules/s3"

  name   = each.key
  config = each.value
  tags   = lookup(local.config, "tags", {})
}

module "eks" {
  source = "./modules/eks"

  config = lookup(local.config, "eks", {})
  tags   = lookup(local.config, "tags", {})
}
```

### Expected resolved output for `environment=prod`, `tenant=acme`

```yaml
tags:
  managed_by: opentofu
  team: platform

databases:
  main:
    host: db.example.internal
    password: (sensitive)
    vpc_id: vpc-12345678
    engine: postgres

eks:
  node_size: m5.xlarge
  min_nodes: 3
  max_nodes: 50

sqs_queues:
  orders:
    retention: 604800
    visibility_timeout: 120
    dlq: true
    dlq_max_retries: 5
  notifications:
    retention: 86400
    dlq: true
    visibility_timeout: 30
  payments:
    retention: 86400
    dlq: true
    visibility_timeout: 30
    dlq_max_retries: 5

dynamodb_tables: {}

s3_buckets: {}
```

---

## 13. Future Considerations (Out of Scope for v1)

These features are explicitly **out of scope** for the initial release but should be considered in the architecture to avoid precluding them:

- **Provider-defined functions**: Expose `provider::confstack::merge()` and `provider::confstack::resolve()` as functions (OpenTofu >= 1.7 / Terraform >= 1.8) for use directly in locals without a data source.
- **Native Secret Decryption**: Direct integration with tools like Mozilla SOPS for automatic decryption of `*.sops.yaml` files.
- **Schema validation**: Optional JSON Schema or OpenTofu type constraint validation of the resolved output.
- **Conditional includes**: A mechanism to conditionally include/exclude files or keys based on feature flags.
- **List merge strategies**: Configurable list merge behavior (replace, append, prepend, merge-by-key).
- **Config diffing**: A CLI or data source that shows what changed between two tenant/environment combinations.

---

## 14. Acceptance Criteria

The provider is considered complete when:

1. All functional requirements (FR-01 through FR-09) pass automated tests.
2. All non-functional requirements (NFR-01 through NFR-08) are met.
3. The provider can be installed from a registry (or local filesystem) and used in a real OpenTofu project.
4. The usage example in Section 12 produces the exact expected output shown.
5. `tofu plan` correctly detects changes when any YAML file in the config directory is modified.
6. Documentation is published and includes at least one complete end-to-end example.
