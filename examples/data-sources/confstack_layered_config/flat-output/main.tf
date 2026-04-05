terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates flat_config for ergonomic HCL attribute access.
# flat_config converts nested keys to dot-separated strings.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
  ]
}

# Access nested values without complex traversal:
# data.confstack_layered_config.example.flat_config["database.host"]
# data.confstack_layered_config.example.flat_config["eks.node_size"]

output "flat_config" {
  description = "All config keys flattened to dot-separated strings"
  value       = data.confstack_layered_config.example.flat_config
}

# Use flat_config to pass specific values to resources
output "database_host" {
  value = data.confstack_layered_config.example.flat_config["database.host"]
}
