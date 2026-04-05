terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

# Build layer list dynamically based on the environment variable.
# Layers that don't exist are silently skipped (on_missing_layer = "skip").
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
    "${path.module}/config/${var.environment}.yaml",
  ]
  on_missing_layer = "skip"
}

output "config" {
  value = data.confstack_layered_config.example.config
}
