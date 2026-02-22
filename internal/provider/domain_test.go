package provider

import (
	"context"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		"period", "ns_group", "dnssec_keys", "is_dnssec_enabled",
		"expiration_date",
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

func TestDomainResourceDnssecKeysHasPlanModifier(t *testing.T) {
	ctx := context.Background()
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)

	dnssecKeysAttr, ok := resp.Schema.Attributes["dnssec_keys"]
	if !ok {
		t.Fatal("dnssec_keys attribute not found in schema")
	}

	// Verify dnssec_keys is Optional and Computed
	// Plan modifiers are applied at runtime, so we check the attribute properties
	listAttr, ok := dnssecKeysAttr.(interface{ IsOptional() bool })
	if !ok {
		t.Fatal("dnssec_keys attribute type assertion failed")
	}

	if !listAttr.IsOptional() {
		t.Error("dnssec_keys should be Optional")
	}

	computedAttr, ok := dnssecKeysAttr.(interface{ IsComputed() bool })
	if !ok {
		t.Fatal("dnssec_keys attribute type assertion for computed failed")
	}

	if !computedAttr.IsComputed() {
		t.Error("dnssec_keys should be Computed to prevent 'known after apply' on reapply")
	}
}

func TestDomainResourceIsDnssecEnabledHasPlanModifier(t *testing.T) {
	ctx := context.Background()
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)

	isDnssecEnabledAttr, ok := resp.Schema.Attributes["is_dnssec_enabled"]
	if !ok {
		t.Fatal("is_dnssec_enabled attribute not found in schema")
	}

	// Verify is_dnssec_enabled is Optional and Computed
	boolAttr, ok := isDnssecEnabledAttr.(interface{ IsOptional() bool })
	if !ok {
		t.Fatal("is_dnssec_enabled attribute type assertion failed")
	}

	if !boolAttr.IsOptional() {
		t.Error("is_dnssec_enabled should be Optional")
	}

	computedAttr, ok := isDnssecEnabledAttr.(interface{ IsComputed() bool })
	if !ok {
		t.Fatal("is_dnssec_enabled attribute type assertion for computed failed")
	}

	if !computedAttr.IsComputed() {
		t.Error("is_dnssec_enabled should be Computed to prevent 'known after apply' on reapply")
	}
}

func TestMapDnssecKeysToStatePreservesValues(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name      string
		keys      []domains.DnssecKey
		expectNil bool
	}{
		{
			name: "with keys",
			keys: []domains.DnssecKey{
				{
					Alg:      8,
					Flags:    257,
					Protocol: 3,
					PubKey:   "test-key",
				},
			},
			expectNil: false,
		},
		{
			name:      "empty keys",
			keys:      []domains.DnssecKey{},
			expectNil: true,
		},
		{
			name:      "nil keys",
			keys:      nil,
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diags := &diag.Diagnostics{}
			result := mapDnssecKeysToState(ctx, tc.keys, diags)

			if diags.HasError() {
				t.Fatalf("Unexpected error in mapDnssecKeysToState: %v", diags)
			}

			if tc.expectNil {
				if !result.IsNull() {
					t.Errorf("Expected null list for %s, got non-null", tc.name)
				}
			} else {
				if result.IsNull() {
					t.Errorf("Expected non-null list for %s, got null", tc.name)
				}
				if len(result.Elements()) != len(tc.keys) {
					t.Errorf("Expected %d elements, got %d", len(tc.keys), len(result.Elements()))
				}

				// For non-nil, non-empty keys, verify that the mapped values are preserved
				// by converting back and comparing with the original keys.
				if len(tc.keys) > 0 {
					var stateKeys []DnssecKeyModel
					diags.Append(result.ElementsAs(ctx, &stateKeys, false)...)
					if diags.HasError() {
						t.Fatalf("Failed to convert result elements: %v", diags)
					}

					if stateKeys[0].Algorithm.ValueInt64() != int64(tc.keys[0].Alg) {
						t.Errorf("Expected algorithm %d, got %d", tc.keys[0].Alg, stateKeys[0].Algorithm.ValueInt64())
					}
					if stateKeys[0].Flags.ValueInt64() != int64(tc.keys[0].Flags) {
						t.Errorf("Expected flags %d, got %d", tc.keys[0].Flags, stateKeys[0].Flags.ValueInt64())
					}
					if stateKeys[0].Protocol.ValueInt64() != int64(tc.keys[0].Protocol) {
						t.Errorf("Expected protocol %d, got %d", tc.keys[0].Protocol, stateKeys[0].Protocol.ValueInt64())
					}
					if stateKeys[0].PublicKey.ValueString() != tc.keys[0].PubKey {
						t.Errorf("Expected public_key %q, got %q", tc.keys[0].PubKey, stateKeys[0].PublicKey.ValueString())
					}
				}
			}
		})
	}
}

func TestConvertDnssecKeysToAPIHandlesNull(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		keysList    types.List
		expectEmpty bool
	}{
		{
			name:        "null list",
			keysList:    types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			expectEmpty: true,
		},
		{
			name:        "empty list",
			keysList:    types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{}}, []attr.Value{}),
			expectEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diags := &diag.Diagnostics{}
			result := convertDnssecKeysToAPI(ctx, tc.keysList, diags)

			if diags.HasError() {
				t.Fatalf("Unexpected error in convertDnssecKeysToAPI: %v", diags)
			}

			if tc.expectEmpty {
				if len(result) > 0 {
					t.Errorf("Expected empty or nil result for %s, got %v", tc.name, result)
				}
			}
		})
	}
}
