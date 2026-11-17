package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func readSSLOrders(t *testing.T, config SSLOrdersModel) SSLOrdersModel {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ssl/orders" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+
			`{"id":1,"common_name":"www.example.com","product_id":7,"brand_name":"Sectigo","status":"ACT","expiration_date":"2027-01-01 00:00:00","autorenew":"on"},`+
			`{"id":2,"common_name":"www.example.com","product_id":7,"status":"EXP","expiration_date":"2025-01-01 00:00:00","autorenew":"off"},`+
			`{"id":3,"common_name":"api.example.com","product_id":8,"status":"ACT"}],"total":3}}`)
	}))
	t.Cleanup(srv.Close)

	d := &SSLOrdersDataSource{client: client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})}
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	config.ID = types.StringUnknown()
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, config); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got SSLOrdersModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	return got
}

func TestSSLOrdersListsAllOrders(t *testing.T) {
	got := readSSLOrders(t, SSLOrdersModel{CommonName: types.StringNull(), Status: types.ListNull(types.StringType)})
	if got.ID.ValueString() != "all" || len(got.Orders) != 3 {
		t.Fatalf("state: got %+v", got)
	}
	first := got.Orders[0]
	if first.ID.ValueInt64() != 1 || first.Status.ValueString() != "ACT" || first.ExpirationDate.ValueString() != "2027-01-01 00:00:00" || !first.Autorenew.ValueBool() {
		t.Errorf("order 1: got %+v", first)
	}
}

func TestSSLOrdersFilters(t *testing.T) {
	status, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"ACT"})
	got := readSSLOrders(t, SSLOrdersModel{CommonName: types.StringValue("www.example.com"), Status: status})
	if got.ID.ValueString() != "www.example.com" || len(got.Orders) != 1 || got.Orders[0].ID.ValueInt64() != 1 {
		t.Errorf("state: got %+v", got)
	}

	got = readSSLOrders(t, SSLOrdersModel{CommonName: types.StringNull(), Status: status})
	if len(got.Orders) != 2 {
		t.Errorf("status filter: got %d orders, want 2", len(got.Orders))
	}
}
