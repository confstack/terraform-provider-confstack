terraform {
  required_providers {
    confstack = {
      source = "confstack/confstack"
    }
  }
}

provider "confstack" {}

# Demonstrates var() and secret() template functions.
# Variables are injected as plain values; secrets become "(sensitive)" in config
# and are available in plaintext only via sensitive_config.
data "confstack_layered_config" "example" {
  layers = [
    "${path.module}/config/base.yaml",
  ]

  variables = {
    VPC_ID       = "vpc-0abc1234"
    CLUSTER_NAME = "my-cluster"
  }

  secrets = {
    DB_PASSWORD = var.db_password
    API_KEY     = var.api_key
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "api_key" {
  type      = string
  sensitive = true
}

output "config" {
  value = data.confstack_layered_config.example.config
}

output "sensitive_config" {
  value     = data.confstack_layered_config.example.sensitive_config
  sensitive = true
}
