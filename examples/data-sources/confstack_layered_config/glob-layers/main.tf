terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates glob pattern support in layers.
# Each layers entry may be a literal path or a glob (including ** for recursion).
# Glob patterns expand to alphabetically sorted concrete paths at their position.
# Later paths still win over earlier ones (last-wins merge).
data "confstack_layered_config" "example" {
  layers = [
    # Explicit base layer — lowest priority
    "${path.module}/config/base.yaml",
    # Glob: all YAML files in overrides/, expanded alphabetically
    "${path.module}/config/overrides/*.yaml",
  ]
}

# The resolved config after merging base + all override files.
output "config" {
  value = data.confstack_layered_config.example.config
}

# Inspect which concrete files were loaded (globs replaced by their matches).
output "loaded_layers" {
  description = "Concrete file paths that were loaded (globs expanded)"
  value       = data.confstack_layered_config.example.loaded_layers
}
