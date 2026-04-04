# Basic Example

This example demonstrates the simplest possible setup for the `confstack` provider. It shows how to load a global baseline and override specific values for a single environment.

## Key Features
- **Global Defaults**: Defined in `_global/defaults.common.yaml`.
- **Environment Overrides**: Defined in `dev/defaults.common.yaml`.
- **Recursive Merge**: The `tags` map is merged across levels.

## How to Run
1. Ensure the provider is built and installed locally.
2. Navigate to this directory: `cd examples/data-sources/confstack_config/basic/`
3. Run `tofu plan` (no variables required, as it defaults to `dev`).

## Expected Output
The `resolved_config` output will show:
- `tags.managed_by` from global.
- `tags.billing` overridden by the dev environment.
- `compute` configuration introduced only in dev.
