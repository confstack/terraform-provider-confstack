<div align="center">

# OpenTofu Provider ConfStack

**OpenTofu provider for managing YAML config GitOps style**

![Terraform](https://img.shields.io/badge/terraform-%235835CC.svg?style=for-the-badge&logo=terraform&logoColor=white)

</div>

---

An OpenTofu/Terraform provider that resolves layered, hierarchical YAML configuration into a single merged output. Designed for multi-environment, multi-tenant infrastructure projects using a flat HCL codebase.

It replaces manual `yamldecode` + deep merge + defaults wiring with a single `confstack_config` data source.

## Features

- **8-level merge priority** across global and environment scopes, with common and tenant variants
- **Deep merge** with strict type checking — maps merge recursively, lists replace, `null` tombstones delete keys
- **Template inheritance** via `_templates` / `_inherit` with bubble-up lookup and `except` lists
- **Go template injection** with `{{ var "KEY" }}` and `{{ secret "KEY" }}` functions (Sprig included)
- **Secret tracking** — secrets are redacted in `output` but available in `sensitive_output`
- **GitOps-safe** — reads exclusively from the local filesystem, no network calls
- **Symlink protection** — files cannot escape `config_dir` via symlinks

## Requirements

- OpenTofu >= 1.6 or Terraform >= 1.6
- Go 1.21+ (to build from source)

## Installation

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

## Directory Convention

```
<config_dir>/
├── _global/                         # Applies to all environments
│   ├── defaults.common.yaml         # Priority 1 — universal baseline
│   ├── defaults.<tenant>.yaml       # Priority 2 — tenant-wide baseline
│   ├── <domain>.common.yaml         # Priority 3 — global domain (all tenants)
│   └── <domain>.<tenant>.yaml       # Priority 4 — global domain (specific tenant)
└── <environment>/                   # Applies to the specified environment
    ├── defaults.common.yaml         # Priority 5 — env baseline
    ├── defaults.<tenant>.yaml       # Priority 6 — env + tenant baseline
    ├── <domain>.common.yaml         # Priority 7 — env domain (all tenants)
    └── <domain>.<tenant>.yaml       # Priority 8 — env domain (specific tenant)
```

Higher priority numbers win. Within the same priority level, files are merged in lexicographic order.

## Usage

```hcl
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
  name     = each.key
  config   = each.value
}
```

### Outputs

| Attribute | Type | Description |
|-----------|------|-------------|
| `output` | dynamic | Fully resolved config map. Secrets shown as `"(sensitive)"`. |
| `sensitive_output` | dynamic (sensitive) | Same map with secrets in plaintext. |
| `loaded_files` | list(string) | Files loaded in merge order. Useful for debugging. |

## YAML Templating

Files are processed as Go templates before YAML parsing:

```yaml
# Non-sensitive: checks variables map, then env vars
vpc_id: {{ var "VPC_ID" }}

# Sensitive: checks secrets map, then env vars — redacted in output
password: {{ secret "DB_PASSWORD" }}

# Built-in context variables
region: {{ if eq .Environment "prod" }}us-east-1{{ else }}us-west-2{{ end }}

# Sprig helpers are available
name: {{ .Tenant | upper }}-cluster
```

## Template Inheritance

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

  # Simple string inheritance
  notifications:
    _inherit: standard
    retention: 3600       # overrides the template value

  # Multiple inheritance with except
  orders:
    _inherit:
      - template: standard
        except: [dlq]     # skip dlq from standard
      - template: critical
    visibility_timeout: 120
```

Templates must have globally unique names. Template entries cannot themselves contain `_inherit`.

## Merge Rules

| Situation | Behavior |
|-----------|----------|
| Map + Map | Recursive deep merge |
| List + List | Full replacement |
| Scalar + Scalar | Full replacement |
| Any + Null | Key deleted (tombstone) |
| Map + Scalar (or vice versa) | **Plan-time error** |

## Security Notice

- `output` is safe to use in plan output — secrets injected via `{{ secret "KEY" }}` are replaced with `"(sensitive)"`.
- `sensitive_output` contains secrets in plaintext and **is stored in the `.tfstate` file**. Restrict access to your state backend accordingly.
- `Sensitive: true` on `sensitive_output` only masks values in CLI plan output; it does **not** encrypt state.

## Building from Source

```sh
git clone https://github.com/confstack/terraform-provider-confstack
cd terraform-provider-confstack
make build

# Install locally for testing
make install
```

## Running Tests

```sh
# Unit tests
make test

# With coverage report
make cover

# E2E tests (requires tofu or terraform in PATH)
make e2e
```

## Development

The project follows clean/hexagonal architecture:

```
internal/
├── domain/          # Core types and error definitions
├── port/
│   ├── input/       # ConfigResolver interface
│   └── output/      # FileDiscoverer, FileReader, YAMLParser, TemplateEngine interfaces
├── usecase/         # Orchestrator + merge, inheritance, cleanup, secrets logic
└── adapter/
    ├── driven/      # Filesystem, YAML, template implementations
    └── driving/     # Terraform provider + data source
```
