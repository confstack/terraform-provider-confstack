provider "confstack" {}

data "confstack_layered_config" "test" {
  layers = [
    "{{CONFIG_DIR}}/_global/defaults.common.yaml",
    "{{CONFIG_DIR}}/prod/compute.acme.yaml",
  ]
}
