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
	if err := os.Setenv("MY_ENV_VAR", "env-value"); err != nil {
		t.Fatalf("failed to set env var MY_ENV_VAR: %v", err)
	}
	if err := os.Setenv("MY_ENV_SECRET", "env-secret"); err != nil {
		t.Fatalf("failed to set env var MY_ENV_SECRET: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MY_ENV_VAR"); err != nil {
			t.Errorf("failed to unset env var MY_ENV_VAR: %v", err)
		}
		if err := os.Unsetenv("MY_ENV_SECRET"); err != nil {
			t.Errorf("failed to unset env var MY_ENV_SECRET: %v", err)
		}
	}()

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
