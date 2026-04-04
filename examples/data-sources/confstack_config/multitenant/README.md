# Multi-Tenant Example

This example demonstrates how to use the `tenant` input to target specific configuration files. It uses the file naming convention `<prefix>.<slug>.yaml`.

## Key Features
- **Tenant Targeting**: Uses the `tenant` variable to pick up `*.acme.yaml` files.
- **Common Baseline**: Loads `*.common.yaml` for all environments/tenants.
- **8-Level Priority**: Merges global common, global tenant, environment common, and environment tenant.

## How to Run
1. Navigate to this directory: `cd examples/data-sources/confstack_config/multitenant/`
2. Run with environment and tenant:
   ```bash
   tofu plan -var="environment=prod" -var="tenant=acme"
   ```

## Expected Output
The `config` output will include:
- Global `databases` configuration.
- Environment-specific `eks` cluster settings for `acme` in `prod`.
- Redacted `password` values from the `secret` function.
