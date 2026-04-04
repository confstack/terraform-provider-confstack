package e2e_test

import (
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driving/terraform"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"confstack": providerserver.NewProtocol6WithError(terraform.New("test")()),
}
