terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates _templates/_inherit for DRY configuration.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
  ]
}

output "config" {
  value = data.confstack_layered_config.example.config
}
