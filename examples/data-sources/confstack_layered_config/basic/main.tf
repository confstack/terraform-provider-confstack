terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Minimal example: two YAML layers merged in order.
# Layer 0 (base) is loaded first; layer 1 overrides any conflicting keys.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
    "${path.module}/config/prod.yaml",
  ]
}

output "config" {
  value = data.confstack_layered_config.example.config
}
