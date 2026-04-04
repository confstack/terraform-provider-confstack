provider "confstack" {}

data "confstack_config" "test" {
  config_dir  = "{{CONFIG_DIR}}"
  environment = "prod"
  tenant      = "acme"
}
