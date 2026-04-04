# Templating Example

This example demonstrates how to use the built-in Go templating engine to inject dynamic data and redact sensitive information.

## Key Features
- **Go Templating**: Each YAML file is processed as a template before parsing.
- **Sprig Helpers**: Uses `upper`, `now`, `date`, and `default`.
- **Variables & Secrets**: Uses `{{ var "KEY" }}` and `{{ secret "KEY" }}`.
- **Multiple Documents**: Uses `---` to separate logical sections in a single file.
- **Granular Sensitivity**: The `output` attribute redacts only the secrets, keeping the rest of the map visible.

## How to Run
1. Navigate to this directory: `cd examples/data-sources/confstack_config/templating/`
2. Run `tofu plan`.

## Expected Output
The `config` output will include:
- `project_info.env_name` as `DEV`.
- `project_info.generated_at` with today's date.
- `infrastructure.vpc_id` as `vpc-abc12345`.
- `infrastructure.db_password` as `(sensitive)`.
