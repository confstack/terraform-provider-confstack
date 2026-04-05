package e2e_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func getFixturePath(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func loadFixtureConfig(t *testing.T, name string) string {
	t.Helper()
	fixturePath := getFixturePath(t, name)
	hclBytes, err := os.ReadFile(filepath.Join(fixturePath, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(hclBytes), "{{CONFIG_DIR}}", filepath.ToSlash(fixturePath))
}

func TestAccLayeredConfigDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: loadFixtureConfig(t, "basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.tags.team", "platform"),
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.eks.node_size", "m5.xlarge"),
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.sqs_queues.orders.retention", "86400"),
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.sqs_queues.orders.dlq", "true"),
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.sqs_queues.orders.visibility_timeout", "120"),
				),
			},
		},
	})
}

func TestAccLayeredConfigDataSource_templating(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: loadFixtureConfig(t, "templating"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.confstack_layered_config.test", "config.db.vpc_id", "vpc-1234"),
				),
			},
		},
	})
}

func TestAccLayeredConfigDataSource_errors(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      loadFixtureConfig(t, "errors"),
				ExpectError: regexp.MustCompile(`merge conflict at path`),
			},
		},
	})
}
