package provider

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestContactDataSourceRead(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{allow: `["192.0.2.1"]`}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	d := &ContactDataSource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "t", HTTPClient: server.Client()})}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	empty := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)

	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: empty}
	cfgState := tfsdk.State{Schema: schemaResp.Schema, Raw: empty}
	if diags := cfgState.Set(ctx, ContactDataSourceModel{
		ID: types.StringValue("7"), Username: types.StringNull(), Email: types.StringNull(), Role: types.StringNull(),
		RCPAllow: types.SetNull(types.StringType), RCPDeny: types.SetNull(types.StringType),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	config.Raw = cfgState.Raw

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: empty}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got ContactDataSourceModel
	_ = resp.State.Get(ctx, &got)
	if got.ID.ValueString() != "7" || !got.RCPAllow.Equal(allowSet([]string{"192.0.2.1"})) || len(got.RCPDeny.Elements()) != 0 {
		t.Errorf("unexpected state %+v", got)
	}
}
