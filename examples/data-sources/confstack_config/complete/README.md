# Complete Example

This example demonstrates the "full resolution power" of the `confstack` provider in a real-world scenario. It uses the resolved configuration to drive multiple infrastructure resources.

## Key Features
- **8-Level Priority**: Combines global shared config, global templates, environment overrides, and tenant-specific settings.
- **Resource Integration**: Shows how to use `for_each` with the resolved config to create multiple resources.
- **`terraform_data`**: Simulates creating SQS queues and S3 buckets with the config.
- **Bubble-up Templates**: Defines templates in `_global/defaults.common.yaml` and uses them in environment-specific files.

## How to Run
1. Navigate to this directory: `cd examples/data-sources/confstack_config/complete/`
2. Run `tofu plan -var="tenant=acme"`.

## Expected Output
The `tofu plan` will show:
- Two `terraform_data.sqs` resources (`notifications`, `payments`) with distinct properties.
- One `terraform_data.s3` resource (`data`) with properties from both the `base` template and the environment override.
- A `summary` output showing the current environment and the resource keys being managed.
