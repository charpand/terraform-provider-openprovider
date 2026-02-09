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

	expectedAttrs := []string{
		"id", "domain", "auth_code", "status", "autorenew",
		"owner_handle", "admin_handle", "tech_handle", "billing_handle",
		"period", "ns_group", "dnssec_keys",
		"import_contacts_from_registry", "import_nameservers_from_registry",
		"is_private_whois_enabled", "expiration_date",
	}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Expected attribute %s not found in schema", attr)
		}
	}

	// Verify auth_code is sensitive
	authCodeAttr := resp.Schema.Attributes["auth_code"]
	if strAttr, ok := authCodeAttr.(interface{ IsSensitive() bool }); ok {
		if !strAttr.IsSensitive() {
			t.Error("auth_code attribute should be sensitive")
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

	expectedAttrs := []string{"id", "domain", "status", "autorenew", "owner_handle", "admin_handle", "tech_handle", "billing_handle", "period"}
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
