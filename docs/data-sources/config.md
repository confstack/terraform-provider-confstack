# confstack_config (Data Source)

Resolves layered, hierarchical YAML configuration into a single merged output object.

## Example Usage

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
```

## Argument Reference

### Required

- `config_dir` (String) — Path to the configuration directory (absolute or relative to the module).
- `environment` (String) — Environment name. Must match a subdirectory in `config_dir` (case-insensitive; normalized to lowercase internally).

### Optional

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `tenant` | String | `""` | Tenant identifier. When empty, only `common` files are loaded. |
| `variables` | Map(String) | `{}` | Variables for `{{ var "KEY" }}` injection. |
| `secrets` | Map(String), Sensitive | `{}` | Sensitive variables for `{{ secret "KEY" }}` injection. |
| `global_dir` | String | `"_global"` | Name of the global scope directory. |
| `common_slug` | String | `"common"` | Slug for files that apply to all tenants. |
| `defaults_prefix` | String | `"defaults"` | Filename prefix for defaults files. |
| `templates_key` | String | `"_templates"` | Reserved YAML key for template definitions. |
| `inherit_key` | String | `"_inherit"` | Reserved YAML key for inheritance directives. |
| `file_extension` | String | `"yaml"` | File extension to match. |

## Attributes Reference

- `output` (Dynamic) — The fully resolved configuration map. Secrets are redacted (shown as `"(sensitive)"`).
- `sensitive_output` (Dynamic, Sensitive) — The fully resolved configuration map with secrets in plaintext.
- `loaded_files` (List(String)) — Ordered list of files that were loaded, in merge priority order. Useful for debugging.

## Template and Inheritance

### `_templates`

Define reusable named templates within any map:

```yaml
sqs_queues:
  _templates:
    standard:
      retention: 86400
      dlq: true
```

Templates are not emitted in the output. Template names must be globally unique.

### `_inherit`

Reference templates using `_inherit`:

```yaml
# Simple string
notifications:
  _inherit: standard
  retention: 3600  # Override a template field

# Multiple templates with except
orders:
  _inherit:
    - template: standard
      except:
        - dlq
    - template: critical
  visibility_timeout: 120
```

Resolution order:
1. Templates are merged left-to-right (each overlays the previous)
2. Entry's own values are applied last as the final override

### Bubble-up Template Lookup

Templates are looked up first in the sibling `_templates` block, then in parent maps up to the root. This allows defining a "global library" of templates in `_global/defaults.common.yaml` that can be referenced anywhere.

## Security Notice

- `output` is safe to use in plan output — secrets injected via `{{ secret "KEY" }}` are replaced with `"(sensitive)"`.
- `sensitive_output` contains secrets in plaintext and **is stored in the `.tfstate` file**. Restrict access to your state backend accordingly.
- `Sensitive: true` on `sensitive_output` only masks values in CLI plan output; it does **not** encrypt state.
