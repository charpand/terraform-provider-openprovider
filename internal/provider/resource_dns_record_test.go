package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func dnsRecordResource(t *testing.T) (*DNSRecordResource, resource.SchemaResponse) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/dns/zones/example.com/records" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+
			`{"name":"example.com","type":"MX","value":"mail1.example.com","ttl":300,"prio":10},`+
			`{"name":"example.com","type":"MX","value":"mail2.example.com","ttl":600,"prio":20},`+
			`{"name":"www.example.com","type":"A","value":"192.0.2.1","ttl":3600}],"total":3}}`)
	}))
	t.Cleanup(srv.Close)
	r := &DNSRecordResource{client: client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)
	return r, sr
}

func importRecord(t *testing.T, id string) (DNSRecordModel, bool) {
	t.Helper()
	ctx := context.Background()
	r, sr := dnsRecordResource(t)
	s := sr.Schema
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
	if resp.Diagnostics.HasError() {
		return DNSRecordModel{}, true
	}
	var got DNSRecordModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	return got, false
}

func TestDNSRecordImportParsesTheIdentifier(t *testing.T) {
	got, errored := importRecord(t, "example.com/www/A")
	if errored {
		t.Fatal("the import should not fail")
	}
	if got.ZoneName.ValueString() != "example.com" || got.Name.ValueString() != "www" || got.Type.ValueString() != "A" {
		t.Errorf("state: got %+v", got)
	}
	if !got.Value.IsNull() {
		t.Errorf("a three part identifier should set no value, got %v", got.Value)
	}
	if got.AllowDeletion.ValueBool() {
		t.Error("an imported record must start with allow_deletion false")
	}
}

func TestDNSRecordImportWithAValue(t *testing.T) {
	// A TXT value can hold slashes; the fourth part keeps them all.
	got, errored := importRecord(t, "example.com/@/TXT/v=spf1 include:a/b -all")
	if errored {
		t.Fatal("the import should not fail")
	}
	if got.Value.ValueString() != "v=spf1 include:a/b -all" {
		t.Errorf("value: got %q", got.Value.ValueString())
	}
}

func TestDNSRecordImportRejectsAnInvalidIdentifier(t *testing.T) {
	for _, id := range []string{"", "example.com", "example.com/www", "example.com//A", "example.com/www/A/"} {
		if _, errored := importRecord(t, id); !errored {
			t.Errorf("identifier %q: the import should fail", id)
		}
	}
}

// TestDNSRecordReadFindsTheRecordOfTheValue guards a zone with two records of
// one name and type: a refresh must read the one that the state holds, not the
// first of the two.
func TestDNSRecordReadFindsTheRecordOfTheValue(t *testing.T) {
	ctx := context.Background()
	r, sr := dnsRecordResource(t)
	s := sr.Schema

	for value, wantTTL := range map[string]int64{"mail2.example.com": 600, "mail1.example.com": 300} {
		m := DNSRecordModel{
			ZoneName: types.StringValue("example.com"), Name: types.StringValue("example.com"), Type: types.StringValue("MX"),
			Value: types.StringValue(value), TTL: types.Int64Value(1), Priority: types.Int64Value(0),
			CreationDate: types.StringValue(""), ModificationDate: types.StringValue(""), ID: types.StringValue("x"), AllowDeletion: types.BoolValue(false),
		}
		state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, m)}
		resp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("read %s: %v", value, resp.Diagnostics)
		}
		var got DNSRecordModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("get: %v", diags)
		}
		if got.Value.ValueString() != value || got.TTL.ValueInt64() != wantTTL {
			t.Errorf("read %s: got value %q ttl %d, want ttl %d", value, got.Value.ValueString(), got.TTL.ValueInt64(), wantTTL)
		}
	}
}

func TestDNSRecordReadAfterAnImportTakesTheFirstRecord(t *testing.T) {
	ctx := context.Background()
	r, sr := dnsRecordResource(t)
	s := sr.Schema

	imported, errored := importRecord(t, "example.com/example.com/MX")
	if errored {
		t.Fatal("the import should not fail")
	}
	imported.TTL, imported.Priority = types.Int64Null(), types.Int64Null()
	state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, imported)}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got DNSRecordModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.Value.ValueString() != "mail1.example.com" || got.Priority.ValueInt64() != 10 || got.ID.ValueString() != "example.com/example.com/MX" {
		t.Errorf("state: got %+v", got)
	}
}
