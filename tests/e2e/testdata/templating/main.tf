provider "confstack" {}

data "confstack_layered_config" "test" {
  layers = [
    "{{CONFIG_DIR}}/_global/defaults.common.yaml",
  ]
  variables = {
    VPC_ID = "vpc-1234"
  }
  secrets = {
    DB_PASSWORD = "supersecret"
  }
}
