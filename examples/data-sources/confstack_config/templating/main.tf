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

  variables = {
    VPC_ID = "vpc-abc12345"
  }

  secrets = {
    DB_PASSWORD = "super-secret-password"
  }
}

output "config" {
  value = data.confstack_config.this.output
}

# The sensitive_output shows secrets in plaintext
output "sensitive_config" {
  value     = data.confstack_config.this.sensitive_output
  sensitive = true
}
