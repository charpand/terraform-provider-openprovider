package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDomainResourceSchema(t *testing.T) {
	ctx := context.Background()
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Schema attributes should not be nil")
	}

	expectedAttrs := []string{"id", "name", "status", "autorenew", "owner_handle", "admin_handle", "tech_handle", "billing_handle", "period"}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Expected attribute %s not found in schema", attr)
		}
	}
}

func TestDomainDataSourceSchema(t *testing.T) {
	ctx := context.Background()
	d := NewDomainDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Schema attributes should not be nil")
	}

	expectedAttrs := []string{"id", "name", "status", "autorenew", "owner_handle", "admin_handle", "tech_handle", "billing_handle", "period"}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Expected attribute %s not found in schema", attr)
		}
	}
}

func TestDomainResourceMetadata(t *testing.T) {
	ctx := context.Background()
	r := NewDomainResource()
	resp := &resource.MetadataResponse{}
	req := resource.MetadataRequest{
		ProviderTypeName: "openprovider",
	}
	r.Metadata(ctx, req, resp)

	expected := "openprovider_domain"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}
}

func TestDomainDataSourceMetadata(t *testing.T) {
	ctx := context.Background()
	d := NewDomainDataSource()
	resp := &datasource.MetadataResponse{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "openprovider",
	}
	d.Metadata(ctx, req, resp)

	expected := "openprovider_domain"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}
}
