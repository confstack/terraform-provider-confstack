package e2e_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLayeredConfigDataSource_metadata(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: loadFixtureConfig(t, "metadata"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.foo", "bar"),
					resource.TestCheckResourceAttrSet("data.confstack_layered_config.test", "loaded_layers.0"),
				),
			},
		},
	})
}

func TestAccLayeredConfigDataSource_envFallback(t *testing.T) {
	os.Setenv("MY_ENV_VAR", "env-value")
	os.Setenv("MY_ENV_SECRET", "env-secret")
	defer os.Unsetenv("MY_ENV_VAR")
	defer os.Unsetenv("MY_ENV_SECRET")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: loadFixtureConfig(t, "env_fallback"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.env_val", "env-value"),
				),
			},
		},
	})
}
