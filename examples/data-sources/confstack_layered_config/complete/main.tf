terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
}

variable "tenant" {
  type        = string
  description = "Tenant identifier. Leave empty to skip the tenant layer."
  default     = ""
}

variable "db_password" {
  type      = string
  sensitive = true
}

# Build a dynamic layer stack:
#   1. Shared base defaults (lowest priority)
#   2. Environment-specific overrides
#   3. Tenant-specific layer (skipped silently if file does not exist)
#
# on_missing_layer = "skip" means any absent file is silently ignored, so
# adding a tenant layer only requires dropping a YAML file — no HCL changes.
data "confstack_layered_config" "example" {
  layers = compact([
    "${path.module}/config/base.yaml",
    "${path.module}/config/${var.environment}.yaml",
    var.tenant != "" ? "${path.module}/config/tenants/${var.tenant}.yaml" : "",
  ])
  on_missing_layer = "skip"

  variables = {
    VPC_ID      = "vpc-0abc1234"
    ENVIRONMENT = var.environment
  }

  secrets = {
    DB_PASSWORD = var.db_password
  }
}

# Nested config object for structured access
output "config" {
  value = data.confstack_layered_config.example.config
}

# Flat view for simple lookups in resource arguments
output "database_host" {
  value = data.confstack_layered_config.example.flat_config["database.host"]
}

output "loaded_layers" {
  description = "Which layer files were actually loaded"
  value       = data.confstack_layered_config.example.loaded_layers
}

output "secret_paths" {
  description = "Paths in config that hold redacted secrets"
  value       = data.confstack_layered_config.example.secret_paths
}
