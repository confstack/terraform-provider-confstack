# confstack Provider

The **confstack** provider resolves layered, hierarchical YAML configuration into a single merged output object. It is designed for multi-environment, multi-tenant infrastructure projects that use a flat HCL codebase with configuration varying by environment and tenant.

## Key Features

- **8-level merge priority**: Global defaults → Global tenant → Global domain → Env defaults → Env tenant → Env domain
- **Deep merge**: Maps merged recursively; lists and scalars replaced; null = tombstone deletion
- **Template inheritance**: `_templates` + `_inherit` with bubble-up resolution and optional `except` lists
- **Go template injection**: `{{ var "KEY" }}` and `{{ secret "KEY" }}` with Sprig helpers
- **Secret tracking**: Secrets are redacted in `output` but available in `sensitive_output`
- **GitOps-safe**: Reads exclusively from the local filesystem; no network calls

## Requirements

- OpenTofu >= 1.6 or Terraform >= 1.6

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
│   ├── defaults.common.yaml         # Priority 1: universal baseline
│   ├── defaults.<tenant>.yaml       # Priority 2: tenant-wide baseline
│   ├── <domain>.common.yaml         # Priority 3: global domain (all tenants)
│   └── <domain>.<tenant>.yaml       # Priority 4: global domain (specific tenant)
└── <environment>/                   # Applies to the specified environment
    ├── defaults.common.yaml         # Priority 5: env baseline
    ├── defaults.<tenant>.yaml       # Priority 6: env+tenant baseline
    ├── <domain>.common.yaml         # Priority 7: env domain (all tenants)
    └── <domain>.<tenant>.yaml       # Priority 8: env domain (specific tenant)
```

**Later entries (higher numbers) take precedence.**

## Merge Rules

| Situation | Behavior |
|-----------|----------|
| Map + Map | Recursive deep merge |
| List + List | Full replacement (no append) |
| Scalar + Scalar | Full replacement |
| Any + Null | Key deleted (tombstone) |
| Null + Any | Key created |
| Map + Scalar (or vice versa) | **Error** |

## Go Templating

Each YAML file is processed as a Go template before parsing:

```yaml
# Use {{ var "KEY" }} for non-sensitive values
vpc_id: {{ var "VPC_ID" }}

# Use {{ secret "KEY" }} for sensitive values (redacted in output)
password: {{ secret "DB_PASSWORD" }}

# Use .Environment and .Tenant in templates
region: {{ if eq .Environment "prod" }}us-east-1{{ else }}us-west-2{{ end }}
```

Both `var` and `secret` check the data source's `variables`/`secrets` inputs first, then fall back to OS environment variables.

## Security Notice

The resolved configuration (including injected secrets) is stored in plaintext in the Terraform/OpenTofu `.tfstate` file. Use `sensitive_output` carefully and restrict access to your state backend.
