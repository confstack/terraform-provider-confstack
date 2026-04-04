# Inheritance Example

This example demonstrates how to use `_templates` and `_inherit` to share common configuration logic across different resources.

## Key Features
- **Global Templates**: Defined in `_global/defaults.common.yaml`.
- **Bubble-up Resolution**: Templates are looked up in sibling and parent maps.
- **Multiple Inheritance**: The `orders` queue inherits from both `standard` and `critical`.
- **Exclusion Lists**: Uses `except` to exclude specific keys from a template.

## How to Run
1. Navigate to this directory: `cd examples/data-sources/confstack_config/inheritance/`
2. Run `tofu plan`.

## Expected Output
The `config` output will show:
- `notifications` queue with `retention` overridden to 3600.
- `orders` queue with `dlq` removed from the `standard` template but present from the `critical` template.
- `visibility_timeout` set to 120 (overridden in the environment).
