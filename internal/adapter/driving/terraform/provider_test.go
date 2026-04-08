package terraform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNewProvider(t *testing.T) {
	factory := New("1.2.3")
	p, ok := factory().(*ConfstackProvider)
	if !ok {
		t.Fatalf("expected *ConfstackProvider, got %T", factory())
	}
	if p.version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", p.version)
	}
}

func TestProviderMetadata(t *testing.T) {
	p := &ConfstackProvider{version: "1.2.3"}
	var resp provider.MetadataResponse

	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)

	if resp.TypeName != "confstack" {
		t.Fatalf("expected type name confstack, got %q", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := &ConfstackProvider{}
	var resp provider.SchemaResponse

	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	if resp.Schema.Description == "" {
		t.Fatal("expected non-empty provider schema description")
	}
}

func TestProviderDataSources(t *testing.T) {
	p := &ConfstackProvider{}

	dataSources := p.DataSources(context.Background())
	if len(dataSources) != 1 {
		t.Fatalf("expected 1 data source, got %d", len(dataSources))
	}

	ds := dataSources[0]()
	if _, ok := ds.(*LayeredConfigDataSource); !ok {
		t.Fatalf("expected *LayeredConfigDataSource, got %T", ds)
	}
}

func TestProviderResources(t *testing.T) {
	p := &ConfstackProvider{}
	if resources := p.Resources(context.Background()); resources != nil {
		t.Fatalf("expected nil resources, got %v", resources)
	}
}

func TestProviderConfigure(t *testing.T) {
	p := &ConfstackProvider{}
	p.Configure(context.Background(), provider.ConfigureRequest{}, &provider.ConfigureResponse{})
}
