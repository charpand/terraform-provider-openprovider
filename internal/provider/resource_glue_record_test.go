package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// glueRecordResource returns a resource wired to a test client for server.
func glueRecordResource(server *httptest.Server) *GlueRecordResource {
	return &GlueRecordResource{client: client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})}
}

// glueRecordStateOf builds a plan or state of the resource's schema from a
// model, the way the framework hands one to Create/Update/Delete.
func glueRecordStateOf(ctx context.Context, t *testing.T, r *GlueRecordResource, m GlueRecordModel) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	if diags := state.Set(ctx, m); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return state
}

func ipsOf(ctx context.Context, t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		t.Fatalf("building ips set: %v", diags)
	}
	return set
}

func TestGlueRecordCreatePublishesTheAddresses(t *testing.T) {
	ctx := context.Background()
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1beta/dns/nameservers" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	res := glueRecordResource(server)
	plan := GlueRecordModel{
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1", "2001:db8::1"),
	}
	resp := &resource.CreateResponse{State: glueRecordStateOf(ctx, t, res, plan)}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: resp.State.Schema, Raw: glueRecordStateOf(ctx, t, res, plan).Raw}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	var got GlueRecordModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "ns1.example.com" {
		t.Errorf("expected id ns1.example.com, got %q", got.ID.ValueString())
	}
	if gotBody != `{"name":"ns1.example.com","ip":"192.0.2.1","ip6":"2001:db8::1"}` {
		t.Errorf("unexpected request body %q", gotBody)
	}
}

func TestGlueRecordCreateRefusesTwoIPv4Addresses(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("expected no request, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	res := glueRecordResource(server)
	plan := GlueRecordModel{
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1", "192.0.2.2"),
	}
	resp := &resource.CreateResponse{State: glueRecordStateOf(ctx, t, res, plan)}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: resp.State.Schema, Raw: glueRecordStateOf(ctx, t, res, plan).Raw}}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected two IPv4 addresses to be refused")
	}
}

func TestGlueRecordCreateRefusesNoAddresses(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("expected no request, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	res := glueRecordResource(server)
	plan := GlueRecordModel{
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t),
	}
	resp := &resource.CreateResponse{State: glueRecordStateOf(ctx, t, res, plan)}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: resp.State.Schema, Raw: glueRecordStateOf(ctx, t, res, plan).Raw}}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a glue record with no addresses to be refused")
	}
}

func TestGlueRecordReadRemovesAMissingRecord(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	res := glueRecordResource(server)
	state := GlueRecordModel{
		ID:        types.StringValue("ns1.example.com"),
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1"),
	}
	priorState := glueRecordStateOf(ctx, t, res, state)
	resp := &resource.ReadResponse{State: priorState}
	res.Read(ctx, resource.ReadRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("expected a 404 to remove the resource from state")
	}
}

func TestGlueRecordReadRefreshesTheAddresses(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/dns/nameservers/ns1.example.com" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"name":"ns1.example.com","ip":"192.0.2.9"}}`))
	}))
	defer server.Close()

	res := glueRecordResource(server)
	state := GlueRecordModel{
		ID:        types.StringValue("ns1.example.com"),
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1"),
	}
	priorState := glueRecordStateOf(ctx, t, res, state)
	resp := &resource.ReadResponse{State: priorState}
	res.Read(ctx, resource.ReadRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got GlueRecordModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	var ips []string
	got.IPs.ElementsAs(ctx, &ips, false)
	if len(ips) != 1 || ips[0] != "192.0.2.9" {
		t.Errorf("expected the refreshed address, got %v", ips)
	}
}

func TestGlueRecordDeleteWithdrawsTheRecord(t *testing.T) {
	ctx := context.Background()
	var deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		deletedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	res := glueRecordResource(server)
	state := GlueRecordModel{
		ID:        types.StringValue("ns1.example.com"),
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1"),
	}
	priorState := glueRecordStateOf(ctx, t, res, state)
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete: %v", resp.Diagnostics)
	}
	if deletedPath != "/v1beta/dns/nameservers/ns1.example.com" {
		t.Errorf("expected a delete for ns1.example.com, got %q", deletedPath)
	}
}

func TestGlueRecordDeleteIsANoOpWhenAlreadyGone(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	res := glueRecordResource(server)
	state := GlueRecordModel{
		ID:        types.StringValue("ns1.example.com"),
		Domain:    types.StringValue("example.com"),
		Subdomain: types.StringValue("ns1"),
		IPs:       ipsOf(ctx, t, "192.0.2.1"),
	}
	priorState := glueRecordStateOf(ctx, t, res, state)
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected deleting an already-gone record not to error, got %v", resp.Diagnostics)
	}
}

func TestGlueRecordImportStateSplitsTheHostName(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("expected no request, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	res := glueRecordResource(server)
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
	}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "ns1.example.com"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	var got GlueRecordModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.Subdomain.ValueString() != "ns1" || got.Domain.ValueString() != "example.com" {
		t.Errorf("expected subdomain ns1 and domain example.com, got %+v", got)
	}
}

func TestGlueRecordImportStateRejectsAnIDWithoutADot(t *testing.T) {
	ctx := context.Background()
	res := &GlueRecordResource{}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
	}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "ns1"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an id without a dot to be rejected")
	}
}
