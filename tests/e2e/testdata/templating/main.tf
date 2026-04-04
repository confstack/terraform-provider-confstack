provider "confstack" {}

data "confstack_config" "test" {
  config_dir  = "{{CONFIG_DIR}}"
  environment = "dev"
  variables = {
    VPC_ID = "vpc-1234"
  }
  secrets = {
    DB_PASSWORD = "supersecret"
  }
}
