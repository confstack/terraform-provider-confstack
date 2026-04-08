package terraform

import (
	"context"
	"fmt"
	"sort"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/filesystem"
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/logging"
	tmplAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/template"
	yamlAdapter "github.com/confstack/terraform-provider-confstack/internal/adapter/driven/yaml"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &LayeredConfigDataSource{}

// LayeredConfigDataSource implements the confstack_layered_config data source.
type LayeredConfigDataSource struct {
	resolver *usecase.Resolver
}

// NewLayeredConfigDataSource returns a new LayeredConfigDataSource factory function.
func NewLayeredConfigDataSource() datasource.DataSource {
	return &LayeredConfigDataSource{}
}

// Metadata sets the type name for the confstack_layered_config data source.
func (d *LayeredConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_layered_config"
}

// Schema defines the attributes accepted and produced by the confstack_layered_config data source.
func (d *LayeredConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Merges an ordered list of YAML layer files into a single configuration map. Last layer wins.",
		Attributes: map[string]schema.Attribute{
			"layers": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Ordered list of YAML file paths (or glob patterns, including ** for recursive matching) to load and merge. Index 0 is lowest priority; last entry is highest. Glob patterns are expanded alphabetically at their position.",
			},
			"on_missing_layer": schema.StringAttribute{
				Optional:    true,
				Description: `How to handle a layer file that does not exist. One of: "error" (default), "warn", "skip".`,
			},
			"variables": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Variables for Go template {{ var \"KEY\" }} injection.",
			},
			"secrets": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "Sensitive variables for Go template {{ secret \"KEY\" }} injection.",
			},
			"flat_separator": schema.StringAttribute{
				Optional:    true,
				Description: `Separator used when flattening nested keys into flat_config. Default: ".".`,
			},
			"config": schema.DynamicAttribute{
				Computed:    true,
				Description: "The fully resolved configuration map (secrets are redacted).",
			},
			"sensitive_config": schema.DynamicAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The fully resolved configuration map with secrets in plaintext.",
			},
			"flat_config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Flattened view of config with separator-delimited keys. All values are converted to strings.",
			},
			"loaded_layers": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ordered list of layer paths that were successfully loaded.",
			},
			"secret_paths": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of flat paths (dot-delimited) that contain secret values.",
			},
		},
	}
}

// layeredConfigDataSourceModel is the Terraform state model for confstack_layered_config.
type layeredConfigDataSourceModel struct {
	Layers          types.List    `tfsdk:"layers"`
	OnMissingLayer    types.String  `tfsdk:"on_missing_layer"`
	Variables       types.Map     `tfsdk:"variables"`
	Secrets         types.Map     `tfsdk:"secrets"`
	FlatSeparator   types.String  `tfsdk:"flat_separator"`
	Config          types.Dynamic `tfsdk:"config"`
	SensitiveConfig types.Dynamic `tfsdk:"sensitive_config"`
	FlatConfig      types.Map     `tfsdk:"flat_config"`
	LoadedLayers    types.List    `tfsdk:"loaded_layers"`
	SecretPaths     types.List    `tfsdk:"secret_paths"`
}

// Configure builds the resolver with real adapters.
func (d *LayeredConfigDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.resolver = usecase.NewResolver(
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewTfLogger(),
		filesystem.NewExpander(),
	)
}

// optionalString returns the string value if set, otherwise empty string.
func optionalString(a types.String) string {
	if a.IsNull() || a.IsUnknown() {
		return ""
	}
	return a.ValueString()
}

// Read resolves the configuration and populates the state attributes.
func (d *LayeredConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state layeredConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract layers list.
	var layers []string
	resp.Diagnostics.Append(state.Layers.ElementsAs(ctx, &layers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract variables and secrets.
	vars := map[string]string{}
	if !state.Variables.IsNull() && !state.Variables.IsUnknown() {
		resp.Diagnostics.Append(state.Variables.ElementsAs(ctx, &vars, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	secrets := map[string]string{}
	if !state.Secrets.IsNull() && !state.Secrets.IsUnknown() {
		resp.Diagnostics.Append(state.Secrets.ElementsAs(ctx, &secrets, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Build functional options.
	opts := []func(*domain.ResolveRequest){
		domain.WithVariables(vars),
		domain.WithSecrets(secrets),
	}
	if ml := optionalString(state.OnMissingLayer); ml != "" {
		opts = append(opts, domain.WithOnMissingLayer(ml))
	}
	if fs := optionalString(state.FlatSeparator); fs != "" {
		opts = append(opts, domain.WithFlatSeparator(fs))
	}

	resolveReq, err := domain.NewResolveRequest(layers, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Invalid layered config request", err.Error())
		return
	}

	result, err := d.resolver.Resolve(ctx, resolveReq)
	if err != nil {
		resp.Diagnostics.AddError("Config resolution failed", err.Error())
		return
	}

	// Convert config to types.Dynamic.
	configVal, err := mapToTerraformDynamic(result.Output)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert config to Terraform value", err.Error())
		return
	}
	state.Config = configVal

	// Convert sensitive_config to types.Dynamic.
	sensConfigVal, err := mapToTerraformDynamic(result.SensitiveOutput)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert sensitive_config to Terraform value", err.Error())
		return
	}
	state.SensitiveConfig = sensConfigVal

	// Convert flat_config to types.Map(string).
	flatElems := make(map[string]attr.Value, len(result.FlatOutput))
	for k, v := range result.FlatOutput {
		flatElems[k] = types.StringValue(fmt.Sprintf("%v", v))
	}
	flatConfigVal, diags := types.MapValue(types.StringType, flatElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.FlatConfig = flatConfigVal

	// Build loaded_layers list.
	loadedElems := make([]attr.Value, len(result.LoadedLayers))
	for i, p := range result.LoadedLayers {
		loadedElems[i] = types.StringValue(p)
	}
	loadedList, diags := types.ListValue(types.StringType, loadedElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.LoadedLayers = loadedList

	// Build secret_paths list (sorted for determinism).
	secretPathSlice := make([]string, 0, len(result.SecretPaths))
	for p := range result.SecretPaths {
		secretPathSlice = append(secretPathSlice, p)
	}
	sort.Strings(secretPathSlice)
	secretElems := make([]attr.Value, len(secretPathSlice))
	for i, p := range secretPathSlice {
		secretElems[i] = types.StringValue(p)
	}
	secretList, diags := types.ListValue(types.StringType, secretElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.SecretPaths = secretList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
