package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestNewLayeredConfigDataSource(t *testing.T) {
	if _, ok := NewLayeredConfigDataSource().(*LayeredConfigDataSource); !ok {
		t.Fatalf("expected *LayeredConfigDataSource, got %T", NewLayeredConfigDataSource())
	}
}

func TestLayeredConfigDataSourceMetadata(t *testing.T) {
	ds := &LayeredConfigDataSource{}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "confstack"}, &resp)

	if resp.TypeName != "confstack_layered_config" {
		t.Fatalf("expected type name confstack_layered_config, got %q", resp.TypeName)
	}
}

func TestLayeredConfigDataSourceSchema(t *testing.T) {
	ds := &LayeredConfigDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Schema.Description == "" {
		t.Fatal("expected non-empty schema description")
	}
	for _, attrName := range []string{"layers", "config", "sensitive_config", "flat_config", "loaded_layers", "secret_paths"} {
		if _, ok := resp.Schema.Attributes[attrName]; !ok {
			t.Fatalf("expected attribute %q in schema", attrName)
		}
	}
}

func TestLayeredConfigDataSourceConfigure(t *testing.T) {
	ds := &LayeredConfigDataSource{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})
	if ds.resolver == nil {
		t.Fatal("expected resolver to be initialized")
	}
}

func TestOptionalString(t *testing.T) {
	if got := optionalString(types.StringNull()); got != "" {
		t.Fatalf("expected empty string for null, got %q", got)
	}
	if got := optionalString(types.StringUnknown()); got != "" {
		t.Fatalf("expected empty string for unknown, got %q", got)
	}
	if got := optionalString(types.StringValue("value")); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestLayeredConfigDataSourceRead_Success(t *testing.T) {
	ctx := context.Background()
	ds := &LayeredConfigDataSource{}
	ds.Configure(ctx, datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})

	dir := t.TempDir()
	layerPath := filepath.Join(dir, "config[prod].yaml")
	if err := os.WriteFile(layerPath, []byte("env: prod\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	schemaResp := schemaResponse(t, ds)
	req := datasource.ReadRequest{
		Config: buildConfig(t, schemaResp.Schema, layeredConfigDataSourceModel{
			Layers:          stringListValue(t, []string{domain.LiteralLayerPrefix + layerPath}),
			OnMissingLayer:  types.StringNull(),
			Variables:       types.MapNull(types.StringType),
			Secrets:         types.MapNull(types.StringType),
			FlatSeparator:   types.StringNull(),
			Config:          types.DynamicNull(),
			SensitiveConfig: types.DynamicNull(),
			FlatConfig:      types.MapNull(types.StringType),
			LoadedLayers:    types.ListNull(types.StringType),
			SecretPaths:     types.ListNull(types.StringType),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var state layeredConfigDataSourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected state diagnostics: %v", resp.Diagnostics)
	}
	if state.LoadedLayers.IsNull() {
		t.Fatal("expected loaded layers to be set")
	}
	var loaded []string
	resp.Diagnostics.Append(state.LoadedLayers.ElementsAs(ctx, &loaded, false)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected loaded layers diagnostics: %v", resp.Diagnostics)
	}
	if len(loaded) != 1 || loaded[0] != layerPath {
		t.Fatalf("expected loaded layer %q, got %v", layerPath, loaded)
	}
}

func TestLayeredConfigDataSourceRead_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	ds := &LayeredConfigDataSource{}
	ds.Configure(ctx, datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})

	schemaResp := schemaResponse(t, ds)
	req := datasource.ReadRequest{
		Config: buildConfig(t, schemaResp.Schema, layeredConfigDataSourceModel{
			Layers:          stringListValue(t, []string{}),
			OnMissingLayer:  types.StringNull(),
			Variables:       types.MapNull(types.StringType),
			Secrets:         types.MapNull(types.StringType),
			FlatSeparator:   types.StringNull(),
			Config:          types.DynamicNull(),
			SensitiveConfig: types.DynamicNull(),
			FlatConfig:      types.MapNull(types.StringType),
			LoadedLayers:    types.ListNull(types.StringType),
			SecretPaths:     types.ListNull(types.StringType),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	ds.Read(ctx, req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid request")
	}
}

func TestLayeredConfigDataSourceRead_ResolutionError(t *testing.T) {
	ctx := context.Background()
	ds := &LayeredConfigDataSource{}
	ds.Configure(ctx, datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})

	schemaResp := schemaResponse(t, ds)
	req := datasource.ReadRequest{
		Config: buildConfig(t, schemaResp.Schema, layeredConfigDataSourceModel{
			Layers:          stringListValue(t, []string{filepath.Join(t.TempDir(), "missing.yaml")}),
			OnMissingLayer:  types.StringValue("error"),
			Variables:       types.MapNull(types.StringType),
			Secrets:         types.MapNull(types.StringType),
			FlatSeparator:   types.StringNull(),
			Config:          types.DynamicNull(),
			SensitiveConfig: types.DynamicNull(),
			FlatConfig:      types.MapNull(types.StringType),
			LoadedLayers:    types.ListNull(types.StringType),
			SecretPaths:     types.ListNull(types.StringType),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	ds.Read(ctx, req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for missing layer")
	}
}

func schemaResponse(t *testing.T, ds *LayeredConfigDataSource) datasource.SchemaResponse {
	t.Helper()
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	return resp
}

func stringListValue(t *testing.T, values []string) types.List {
	t.Helper()
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	list, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("unexpected list diagnostics: %v", diags)
	}
	return list
}

func buildConfig(t *testing.T, schema datasourceschema.Schema, model layeredConfigDataSourceModel) tfsdk.Config {
	t.Helper()
	objectType, ok := schema.Type().(basetypes.ObjectType)
	if !ok {
		t.Fatalf("expected object type, got %T", schema.Type())
	}
	values := map[string]attr.Value{
		"layers":           model.Layers,
		"on_missing_layer": model.OnMissingLayer,
		"variables":        model.Variables,
		"secrets":          model.Secrets,
		"flat_separator":   model.FlatSeparator,
		"config":           model.Config,
		"sensitive_config": model.SensitiveConfig,
		"flat_config":      model.FlatConfig,
		"loaded_layers":    model.LoadedLayers,
		"secret_paths":     model.SecretPaths,
	}
	obj, diags := types.ObjectValue(objectType.AttrTypes, values)
	if diags.HasError() {
		t.Fatalf("unexpected object diagnostics: %v", diags)
	}
	raw, err := obj.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("unexpected terraform value error: %v", err)
	}
	return tfsdk.Config{
		Raw:    raw,
		Schema: schema,
	}
}
