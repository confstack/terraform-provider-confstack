package terraform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &ConfstackProvider{}

// ConfstackProvider is the Terraform provider implementation. It has no configuration of its own.
type ConfstackProvider struct {
	version string
}

// New returns a provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ConfstackProvider{version: version}
	}
}

// Metadata sets the provider type name and version.
func (p *ConfstackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "confstack"
	resp.Version = p.version
}

// Schema returns the provider schema (no configuration attributes required).
func (p *ConfstackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The confstack provider resolves layered, hierarchical YAML configuration into a single merged output.",
	}
}

// Configure is a no-op: the confstack provider has no provider-level configuration.
func (p *ConfstackProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
	// No provider-level configuration required.
}

// DataSources returns the list of data sources exposed by this provider.
func (p *ConfstackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewConfigDataSource,
	}
}

// Resources returns nil — this provider exposes no managed resources.
func (p *ConfstackProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
