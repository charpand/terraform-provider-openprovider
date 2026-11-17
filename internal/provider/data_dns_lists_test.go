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

func dnsListAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/dns/zones":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+
				`{"name":"example","extension":"com","type":"master","provider":"openprovider","creation_date":"2026-01-01"},`+
				`{"name":"example","extension":"nl","type":"slave"}],"total":2}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/dns/zones/example.com/records":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+
				`{"name":"www.example.com","type":"A","value":"192.0.2.1","ttl":3600},`+
				`{"name":"example.com","type":"MX","value":"mail.example.com","ttl":300,"prio":10}],"total":2}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dnsListClient(srv *httptest.Server) *client.Client {
	return client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})
}

func TestDNSZonesDataSourceListsTheZones(t *testing.T) {
	ctx := context.Background()
	d := &DNSZonesDataSource{client: dnsListClient(dnsListAPI(t))}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got DNSZonesModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "all" || len(got.Zones) != 2 {
		t.Fatalf("state: got %+v", got)
	}
	if z := got.Zones[0]; z.ZoneName.ValueString() != "example.com" || z.Type.ValueString() != "master" || z.Provider.ValueString() != "openprovider" {
		t.Errorf("zone 0: got %+v", z)
	}
	if got.Zones[1].ZoneName.ValueString() != "example.nl" {
		t.Errorf("zone 1: got %+v", got.Zones[1])
	}
}

func TestDNSRecordsDataSourceListsTheRecords(t *testing.T) {
	ctx := context.Background()
	d := &DNSRecordsDataSource{client: dnsListClient(dnsListAPI(t))}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, DNSRecordsModel{ID: types.StringUnknown(), ZoneName: types.StringValue("example.com"), Records: nil}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got DNSRecordsModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "example.com" || len(got.Records) != 2 {
		t.Fatalf("state: got %+v", got)
	}
	if r := got.Records[1]; r.Type.ValueString() != "MX" || r.Priority.ValueInt64() != 10 || r.TTL.ValueInt64() != 300 {
		t.Errorf("record 1: got %+v", r)
	}
}
