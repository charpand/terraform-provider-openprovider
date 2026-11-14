package provider

import (
	"context"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// transferWithWriteOnlyCode is the model of a transfer whose code is in
// `auth_code_wo`.
func transferWithWriteOnlyCode(code types.String) DomainModel {
	return DomainModel{
		ID:              types.StringUnknown(),
		Domain:          types.StringValue("example.com"),
		OwnerHandle:     types.StringValue("XX000001-NL"),
		AuthCodeWO:      code,
		Autorenew:       types.BoolValue(false),
		Period:          types.Int64Unknown(),
		Status:          types.StringUnknown(),
		DnssecKeys:      noDnssecKeys(),
		AdditionalData:  types.MapNull(types.StringType),
		IsDnssecEnabled: types.BoolUnknown(),
		OnDestroy:       types.StringValue(onDestroyRetain),
	}
}

// TestDomainResourceSchemaAuthCodeWOIsWriteOnly covers the schema: Terraform
// keeps no value of `auth_code_wo` in the plan or in the state.
func TestDomainResourceSchemaAuthCodeWOIsWriteOnly(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewDomainResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	attr, ok := resp.Schema.Attributes["auth_code_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatal("auth_code_wo should be a StringAttribute")
	}
	if !attr.WriteOnly || !attr.Sensitive {
		t.Errorf("auth_code_wo: write-only %t, sensitive %t, want both", attr.WriteOnly, attr.Sensitive)
	}
}

// TestDomainResourceCreateTransfersWithAWriteOnlyCode covers a transfer whose
// code is only in the configuration, as a write-only argument gives it. The
// plan has no code. The provider sends the transfer with the code of the
// configuration, and the state has no code.
func TestDomainResourceCreateTransfersWithAWriteOnlyCode(t *testing.T) {
	ctx := context.Background()
	api := newOrderAPI(t, "example", "com", domainStatusRequested)
	defer api.Close()
	r := &DomainResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})}

	config := stateOf(ctx, t, r, transferWithWriteOnlyCode(types.StringValue("secret")))
	plan := stateOf(ctx, t, r, transferWithWriteOnlyCode(types.StringNull()))
	resp := resource.CreateResponse{State: plan}
	r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config(config), Plan: tfsdk.Plan(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if api.ordered != "/v1/domains/transfer" {
		t.Fatalf("order sent to %q, want the transfer", api.ordered)
	}
	if sent := api.order["auth_code"]; sent != "secret" {
		t.Errorf("the transfer sent the auth code %v, want the code of the configuration", sent)
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if !got.AuthCodeWO.IsNull() || !got.AuthCode.IsNull() {
		t.Errorf("the state keeps an auth code: auth_code_wo %v, auth_code %v", got.AuthCodeWO, got.AuthCode)
	}
}

// TestAuthCodeWOConflictsWithAuthCode covers a configuration that sets both
// `auth_code` and `auth_code_wo`: the validation fails.
func TestAuthCodeWOConflictsWithAuthCode(t *testing.T) {
	ctx := context.Background()
	r := &DomainResource{}
	for name, tc := range map[string]struct {
		authCode types.String
		wantErr  bool
	}{
		"only auth_code_wo": {types.StringNull(), false},
		"both":              {types.StringValue("other"), true},
	} {
		t.Run(name, func(t *testing.T) {
			model := transferWithWriteOnlyCode(types.StringValue("secret"))
			model.AuthCode = tc.authCode
			config := stateOf(ctx, t, r, model)
			resp := &validator.StringResponse{}
			authCodeConflictValidator{}.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("auth_code_wo"),
				ConfigValue: model.AuthCodeWO,
				Config:      tfsdk.Config(config),
			}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
