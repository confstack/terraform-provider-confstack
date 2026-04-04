terraform {
  required_providers {
    confstack = {
      source  = "confstack/confstack"
      version = "~> 1.0"
    }
  }
}

provider "confstack" {}

variable "tenant" {
  type    = string
  default = "acme"
}

data "confstack_config" "this" {
  config_dir  = "${path.module}/config"
  environment = "prod"
  tenant      = var.tenant
}

locals {
  config = data.confstack_config.this.output
}

# Example: Using the resolved config to drive resources
# In a real setup, these would be 'aws_sqs_queue' or similar
resource "terraform_data" "sqs" {
  for_each = lookup(local.config, "sqs_queues", {})

  input = {
    name               = each.key
    retention          = each.value.retention
    dlq                = each.value.dlq
    visibility_timeout = each.value.visibility_timeout
    tags               = local.config.tags
  }
}

resource "terraform_data" "s3" {
  for_each = lookup(local.config, "s3_buckets", {})

  input = {
    name       = each.key
    encryption = each.value.encryption
    versioning = each.value.versioning
    tags       = local.config.tags
  }
}

output "summary" {
  value = {
    env     = local.config.tags.environment
    queues  = keys(local.config.sqs_queues)
    buckets = keys(local.config.s3_buckets)
  }
}
