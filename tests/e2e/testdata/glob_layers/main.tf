provider "confstack" {}

data "confstack_layered_config" "test" {
  layers = [
    "{{CONFIG_DIR}}/base.yaml",
    "{{CONFIG_DIR}}/overrides/*.yaml",
  ]
}
