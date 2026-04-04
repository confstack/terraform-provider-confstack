terraform {
  required_providers {
    confstack = {
      source  = "confstack/confstack"
      version = "~> 1.0"
    }
  }
}

provider "confstack" {}

data "confstack_config" "this" {
  config_dir  = "${path.module}/config"
  environment = "dev"
}

output "resolved_config" {
  value = data.confstack_config.this.output
}

output "loaded_files" {
  value = data.confstack_config.this.loaded_files
}
