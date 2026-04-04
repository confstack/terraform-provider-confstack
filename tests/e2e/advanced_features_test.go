package e2e_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigDataSource_metadata(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: loadFixtureConfig(t, "metadata"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.confstack_config.test", "output.foo", "bar"),
					// FR-07: Verify loaded_files attribute
					resource.TestCheckResourceAttrSet("data.confstack_config.test", "loaded_files.0"),
				),
			},
		},
	})
}

func TestAccConfigDataSource_envFallback(t *testing.T) {
	// FR-02: Verify OS environment variable fallback
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
					resource.TestCheckResourceAttr("data.confstack_config.test", "output.env_val", "env-value"),
				),
			},
		},
	})
}
