package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func nsGroupResource(server *httptest.Server) *NSGroupResource {
	return &NSGroupResource{client: client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})}
}

func nsGroupStateOf(ctx context.Context, t *testing.T, r *NSGroupResource, m NSGroupResourceModel) tfsdk.State {
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

// TestNSGroupCreateReadsBackTheGroup covers the fix in #102: the create
// response carries only a success flag, so Create reads the group back to
// fill in the computed attributes.
func TestNSGroupCreateReadsBackTheGroup(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1beta/dns/nameservers/groups":
			_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/dns/nameservers/groups/my-group":
			_, _ = w.Write([]byte(`{"code":0,"data":{"ns_group":"my-group","name_servers":[{"name":"ns1.example.com","ip":"192.0.2.1"}]}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	res := nsGroupResource(server)
	plan := NSGroupResourceModel{
		Name:          types.StringValue("my-group"),
		Nameservers:   []NSGroupNameserverModel{{Name: types.StringValue("ns1.example.com"), IP: types.StringUnknown(), IP6: types.StringUnknown()}},
		AllowDeletion: types.BoolValue(false),
	}
	resp := &resource.CreateResponse{State: nsGroupStateOf(ctx, t, res, plan)}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: resp.State.Schema, Raw: nsGroupStateOf(ctx, t, res, plan).Raw}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	var got NSGroupResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "my-group" || len(got.Nameservers) != 1 || got.Nameservers[0].IP.ValueString() != "192.0.2.1" {
		t.Errorf("expected the read-back group, got %+v", got)
	}
}

// TestNSGroupReadRemovesAMissingGroup is the regression test for the
// #110-class bug: a group deleted outside Terraform must be dropped from
// state, not surfaced as an error.
func TestNSGroupReadRemovesAMissingGroup(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	res := nsGroupResource(server)
	state := NSGroupResourceModel{
		ID:            types.StringValue("gone-group"),
		Name:          types.StringValue("gone-group"),
		Nameservers:   []NSGroupNameserverModel{{Name: types.StringValue("ns1.example.com"), IP: types.StringNull(), IP6: types.StringNull()}},
		AllowDeletion: types.BoolValue(false),
	}
	priorState := nsGroupStateOf(ctx, t, res, state)
	resp := &resource.ReadResponse{State: priorState}
	res.Read(ctx, resource.ReadRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected a 404 to remove the resource without error, got %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("expected a missing group to be removed from state")
	}
}

func TestNSGroupDeleteKeepsTheGroupByDefault(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("expected no request when allow_deletion is false, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	res := nsGroupResource(server)
	state := NSGroupResourceModel{
		ID:            types.StringValue("my-group"),
		Name:          types.StringValue("my-group"),
		AllowDeletion: types.BoolValue(false),
	}
	priorState := nsGroupStateOf(ctx, t, res, state)
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)
	if resp.Diagnostics.ErrorsCount() > 0 {
		t.Fatalf("expected only a warning, got errors: %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Fatal("expected a warning that the group was kept")
	}
}

func TestNSGroupDeleteDeletesWhenAllowed(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
	}))
	defer server.Close()

	res := nsGroupResource(server)
	state := NSGroupResourceModel{
		ID:            types.StringValue("my-group"),
		Name:          types.StringValue("my-group"),
		AllowDeletion: types.BoolValue(true),
	}
	priorState := nsGroupStateOf(ctx, t, res, state)
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete: %v", resp.Diagnostics)
	}
	if deletedPath != "/v1beta/dns/nameservers/groups/my-group" {
		t.Errorf("expected a delete for my-group, got %q", deletedPath)
	}
}
