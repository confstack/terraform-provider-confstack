terraform {
  required_providers {
    confstack = {
      source  = "confstack/confstack"
      version = "~> 1.0"
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
  description = "Tenant identifier"
  default     = ""
}

variable "db_password" {
  type      = string
  sensitive = true
  default   = "changeme"
}

data "confstack_config" "this" {
  config_dir  = "${path.module}/config"
  environment = var.environment
  tenant      = var.tenant

  variables = {
    VPC_ID = "vpc-12345678"
  }

  secrets = {
    DB_PASSWORD = var.db_password
  }
}

output "config" {
  value = data.confstack_config.this.output
}

output "loaded_files" {
  value = data.confstack_config.this.loaded_files
}
