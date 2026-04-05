---
page_title: "confstack_layered_config Data Source - confstack"
description: |-
  Merges an ordered list of YAML layer files into a single configuration map.
  Supports recursive deep merge, Go template injection for variables and secrets,
  and template inheritance for DRY configuration patterns.
---

# confstack_layered_config (Data Source)

Merges an ordered list of YAML layer files into a single configuration map. The last layer wins on conflicts. Supports Go template injection (`var`/`secret`) and template inheritance (`_templates`/`_inherit`).

## Basic Example

Two YAML files merged in order — `base.yaml` provides defaults, `prod.yaml` overrides specific values.

```terraform
terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Minimal example: two YAML layers merged in order.
# Layer 0 (base) is loaded first; layer 1 overrides any conflicting keys.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
    "${path.module}/config/prod.yaml",
  ]
}

output "config" {
  value = data.confstack_layered_config.example.config
}
```

`config/base.yaml`:

```yaml
tags:
  managed_by: opentofu
  team: platform

eks:
  node_size: t3.medium
  min_nodes: 2
  max_nodes: 10
```

`config/prod.yaml`:

```yaml
tags:
  environment: prod

eks:
  node_size: m5.xlarge
  min_nodes: 3
  max_nodes: 50
```

Result: `config.eks.node_size = "m5.xlarge"`, `config.tags.managed_by = "opentofu"` (kept from base), `config.tags.environment = "prod"` (added by prod layer).

## Variables and Secrets

Inject runtime values into YAML layers using Go template functions. Variables appear as-is in `config`; secrets are redacted as `"(sensitive)"` and only available in plaintext via `sensitive_config`.

```terraform
terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates var() and secret() template functions.
# Variables are injected as plain values; secrets become "(sensitive)" in config
# and are available in plaintext only via sensitive_config.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
  ]

  variables = {
    VPC_ID       = "vpc-0abc1234"
    CLUSTER_NAME = "my-cluster"
  }

  secrets = {
    DB_PASSWORD = var.db_password
    API_KEY     = var.api_key
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "api_key" {
  type      = string
  sensitive = true
}

output "config" {
  value = data.confstack_layered_config.example.config
}

output "sensitive_config" {
  value     = data.confstack_layered_config.example.sensitive_config
  sensitive = true
}
```

`config/base.yaml`:

```yaml
network:
  vpc_id: {{ var "VPC_ID" }}
  cluster: {{ var "CLUSTER_NAME" }}

database:
  password: {{ secret "DB_PASSWORD" }}

integrations:
  api_key: {{ secret "API_KEY" }}
```

-> Sprig's `env "KEY"` also reads OS environment variables but returns an empty string on missing keys (no error) and does not check `variables`. Prefer `var "KEY"` for consistent error handling.

## Inheritance (`_templates` / `_inherit`)

Define reusable config blocks and apply them by name. Useful for sharing base attributes across similar resources without repeating yourself.

```terraform
terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates _templates/_inherit for DRY configuration.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
  ]
}

output "config" {
  value = data.confstack_layered_config.example.config
}
```

`config/base.yaml`:

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

  # Single inheritance: notifications gets all fields from standard,
  # then overrides retention to 3600.
  notifications:
    _inherit: standard
    retention: 3600

  # Multi-inheritance with exception: orders gets standard (minus dlq),
  # then all of critical on top.
  orders:
    _inherit:
      - template: standard
        except:
          - dlq
      - template: critical
```

Rules:
- `_inherit` accepts a string (single name), a list of strings, or a list of `{template, except}` objects to selectively exclude keys.
- Templates are collected globally from all loaded layers and must have unique names.
- Templates cannot themselves contain `_inherit`.
- Both `_templates` and `_inherit` are stripped from all outputs.

## Dynamic Multi-Layer Stack

Use `compact()` and `on_missing_layer = "skip"` to build optional layer stacks. Adding a new environment or tenant only requires a YAML file — no HCL changes.

```terraform
terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
}

variable "tenant" {
  type        = string
  description = "Tenant identifier. Leave empty to skip the tenant layer."
  default     = ""
}

variable "db_password" {
  type      = string
  sensitive = true
}

# Build a dynamic layer stack:
#   1. Shared base defaults (lowest priority)
#   2. Environment-specific overrides
#   3. Tenant-specific layer (skipped silently if file does not exist)
#
# on_missing_layer = "skip" means any absent file is silently ignored, so
# adding a tenant layer only requires dropping a YAML file — no HCL changes.
data "confstack_layered_config" "example" {
  layers = compact([
    "${path.module}/config/base.yaml",
    "${path.module}/config/${var.environment}.yaml",
    var.tenant != "" ? "${path.module}/config/tenants/${var.tenant}.yaml" : "",
  ])
  on_missing_layer = "skip"

  variables = {
    VPC_ID      = "vpc-0abc1234"
    ENVIRONMENT = var.environment
  }

  secrets = {
    DB_PASSWORD = var.db_password
  }
}

# Nested config object for structured access
output "config" {
  value = data.confstack_layered_config.example.config
}

# Flat view for simple lookups in resource arguments
output "database_host" {
  value = data.confstack_layered_config.example.flat_config["database.host"]
}

output "loaded_layers" {
  description = "Which layer files were actually loaded"
  value       = data.confstack_layered_config.example.loaded_layers
}

output "secret_paths" {
  description = "Paths in config that hold redacted secrets"
  value       = data.confstack_layered_config.example.secret_paths
}
```

`config/base.yaml`:

```yaml
tags:
  managed_by: opentofu
  team: platform

database:
  host: db.example.internal
  port: 5432
  password: {{ secret "DB_PASSWORD" }}

network:
  vpc_id: {{ var "VPC_ID" }}

eks:
  node_size: t3.medium
  min_nodes: 2
  max_nodes: 10
```

`config/prod.yaml`:

```yaml
tags:
  environment: prod

eks:
  node_size: m5.xlarge
  min_nodes: 3
  max_nodes: 50

database:
  multi_az: true
  instance_class: db.r5.large
```

`config/tenants/acme.yaml`:

```yaml
tags:
  tenant: acme

eks:
  min_nodes: 5

database:
  name: acme_production
```

## Merge Behavior

Layers are processed in index order — **index 0 is the base (lowest priority), the last entry wins**.

| Value type | Merge strategy |
|---|---|
| Map | Recursively merged; new keys added, existing keys overridden |
| Scalar (string, number, bool) | Replaced entirely by the higher-priority layer |
| List | Replaced entirely (not concatenated) |
| `null` | Deletes the key from the result |

Type mismatches between layers (e.g., a map in one layer and a scalar in another at the same path) produce an error.

## Flat Output

`flat_config` collapses all nested keys into a `map(string)` using `flat_separator` (default `.`). All values are converted to strings. Useful for passing individual config values directly to resource arguments without attribute traversal.

```hcl
resource "aws_db_instance" "main" {
  address = data.confstack_layered_config.app.flat_config["database.host"]
  port    = data.confstack_layered_config.app.flat_config["database.port"]
}
```

## Missing Layer Handling

| `on_missing_layer` | Behavior when a layer file is absent |
|---|---|
| `"error"` (default) | Plan/apply fails immediately. |
| `"warn"` | Warning logged; layer skipped; resolution continues. |
| `"skip"` | Layer silently skipped; resolution continues. |

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `layers` (List of String) Ordered list of YAML file paths to load and merge. Index 0 is lowest priority; last entry is highest.

### Optional

- `flat_separator` (String) Separator used when flattening nested keys into flat_config. Default: ".".
- `on_missing_layer` (String) How to handle a layer file that does not exist. One of: "error" (default), "warn", "skip".
- `secrets` (Map of String, Sensitive) Sensitive variables for Go template {{ secret "KEY" }} injection.
- `variables` (Map of String) Variables for Go template {{ var "KEY" }} injection.

### Read-Only

- `config` (Dynamic) The fully resolved configuration map (secrets are redacted).
- `flat_config` (Map of String) Flattened view of config with separator-delimited keys. All values are converted to strings.
- `loaded_layers` (List of String) Ordered list of layer paths that were successfully loaded.
- `secret_paths` (List of String) List of flat paths (dot-delimited) that contain secret values.
- `sensitive_config` (Dynamic, Sensitive) The fully resolved configuration map with secrets in plaintext.
