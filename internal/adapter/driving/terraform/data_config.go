package terraform

import (
	"context"

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

var _ datasource.DataSource = &ConfigDataSource{}

// ConfigDataSource implements the confstack_config data source.
type ConfigDataSource struct {
	resolver *usecase.Resolver
}

// NewConfigDataSource returns a new ConfigDataSource factory function.
func NewConfigDataSource() datasource.DataSource {
	return &ConfigDataSource{}
}

// Metadata sets the type name for the confstack_config data source.
func (d *ConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

// Schema defines the attributes accepted and produced by the confstack_config data source.
func (d *ConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resolves layered, hierarchical YAML configuration into a single merged output.",
		Attributes: map[string]schema.Attribute{
			"config_dir": schema.StringAttribute{
				Required:    true,
				Description: "Path to the configuration directory (absolute or relative to the module).",
			},
			"environment": schema.StringAttribute{
				Required:    true,
				Description: "Environment name. Must match a subdirectory in config_dir.",
			},
			"tenant": schema.StringAttribute{
				Optional:    true,
				Description: "Tenant identifier. Omit to load only common files.",
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
			"global_dir": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the global scope directory. Default: \"_global\".",
			},
			"common_slug": schema.StringAttribute{
				Optional:    true,
				Description: "Slug used for files that apply to all tenants. Default: \"common\".",
			},
			"defaults_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Filename prefix for defaults files. Default: \"defaults\".",
			},
			"templates_key": schema.StringAttribute{
				Optional:    true,
				Description: "Reserved YAML key for template definitions. Default: \"_templates\".",
			},
			"inherit_key": schema.StringAttribute{
				Optional:    true,
				Description: "Reserved YAML key for inheritance directives. Default: \"_inherit\".",
			},
			"file_extension": schema.StringAttribute{
				Optional:    true,
				Description: "File extension to match. Default: \"yaml\".",
			},
			"output": schema.DynamicAttribute{
				Computed:    true,
				Description: "The fully resolved configuration map (secrets are redacted).",
			},
			"sensitive_output": schema.DynamicAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The fully resolved configuration map with secrets in plaintext.",
			},
			"loaded_files": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ordered list of files that were loaded, in merge priority order.",
			},
		},
	}
}

// configDataSourceModel is the Terraform state model for confstack_config.
type configDataSourceModel struct {
	ConfigDir       types.String  `tfsdk:"config_dir"`
	Environment     types.String  `tfsdk:"environment"`
	Tenant          types.String  `tfsdk:"tenant"`
	Variables       types.Map     `tfsdk:"variables"`
	Secrets         types.Map     `tfsdk:"secrets"`
	GlobalDir       types.String  `tfsdk:"global_dir"`
	CommonSlug      types.String  `tfsdk:"common_slug"`
	DefaultsPrefix  types.String  `tfsdk:"defaults_prefix"`
	TemplatesKey    types.String  `tfsdk:"templates_key"`
	InheritKey      types.String  `tfsdk:"inherit_key"`
	FileExtension   types.String  `tfsdk:"file_extension"`
	Output          types.Dynamic `tfsdk:"output"`
	SensitiveOutput types.Dynamic `tfsdk:"sensitive_output"`
	LoadedFiles     types.List    `tfsdk:"loaded_files"`
}

// Configure builds the resolver with real adapters. This happens once per data source configure call.
func (d *ConfigDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.resolver = usecase.NewResolver(
		filesystem.NewDiscoverer(),
		filesystem.NewReader(),
		yamlAdapter.NewParser(),
		tmplAdapter.NewEngine(),
		logging.NewTfLogger(),
	)
}

// optionalString returns the string value if the attribute is set, otherwise empty string.
func optionalString(attr types.String) string {
	if attr.IsNull() || attr.IsUnknown() {
		return ""
	}
	return attr.ValueString()
}

// Read resolves the configuration and populates the state attributes.
func (d *ConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state configDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract variables and secrets first (diagnostic operations)
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

	// Build optional overrides
	opts := []func(*domain.ResolveRequest){
		domain.WithVariables(vars),
		domain.WithSecrets(secrets),
	}
	if t := optionalString(state.Tenant); t != "" {
		opts = append(opts, domain.WithTenant(t))
	}
	if gd := optionalString(state.GlobalDir); gd != "" {
		opts = append(opts, domain.WithGlobalDir(gd))
	}
	if cs := optionalString(state.CommonSlug); cs != "" {
		opts = append(opts, domain.WithCommonSlug(cs))
	}
	if dp := optionalString(state.DefaultsPrefix); dp != "" {
		opts = append(opts, domain.WithDefaultsPrefix(dp))
	}
	if tk := optionalString(state.TemplatesKey); tk != "" {
		opts = append(opts, domain.WithTemplatesKey(tk))
	}
	if ik := optionalString(state.InheritKey); ik != "" {
		opts = append(opts, domain.WithInheritKey(ik))
	}
	if fe := optionalString(state.FileExtension); fe != "" {
		opts = append(opts, domain.WithFileExtension(fe))
	}

	resolveReq, err := domain.NewResolveRequest(
		state.ConfigDir.ValueString(),
		state.Environment.ValueString(),
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Invalid config resolution request", err.Error())
		return
	}

	result, err := d.resolver.Resolve(ctx, resolveReq)
	if err != nil {
		resp.Diagnostics.AddError("Config resolution failed", err.Error())
		return
	}

	// Convert output map to types.Dynamic
	outputVal, err := mapToTerraformDynamic(result.Output)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert output to Terraform value", err.Error())
		return
	}
	state.Output = outputVal

	sensOutputVal, err := mapToTerraformDynamic(result.SensitiveOutput)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert sensitive_output to Terraform value", err.Error())
		return
	}
	state.SensitiveOutput = sensOutputVal

	// Build loaded_files list
	loadedFileElems := make([]attr.Value, len(result.LoadedFiles))
	for i, f := range result.LoadedFiles {
		loadedFileElems[i] = types.StringValue(f)
	}
	loadedFilesList, diags := types.ListValue(types.StringType, loadedFileElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.LoadedFiles = loadedFilesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
