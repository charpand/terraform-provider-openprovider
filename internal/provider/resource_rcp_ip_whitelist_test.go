package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeContact serves GET and PUT for contact 7 and records the PUT bodies.
type fakeContact struct {
	mu     sync.Mutex
	allow  string // JSON array served on GET; "" serves a contact without list
	puts   []string
	status int // GET status override
}

func (f *fakeContact) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1beta/contacts/7" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			f.puts = append(f.puts, string(body))
			_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
		case http.MethodGet:
			if f.status != 0 {
				w.WriteHeader(f.status)
				_, _ = w.Write([]byte(`{"code":4}`))
				return
			}
			if f.allow == "" {
				_, _ = w.Write([]byte(`{"code":0,"data":{"id":7}}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"id":7,"rcp_client_ip_list":{"allow":` + f.allow + `}}}`))
		}
	}
}

func whitelistResource(server *httptest.Server) *RCPIPWhitelistResource {
	return &RCPIPWhitelistResource{client: client.NewClient(client.Config{
		BaseURL: server.URL, Token: "test", HTTPClient: server.Client(),
	})}
}

func whitelistState(ctx context.Context, t *testing.T, r *RCPIPWhitelistResource, m RCPIPWhitelistResourceModel) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	if diags := state.Set(ctx, m); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return state
}

func whitelistModel(onDestroy string, allow ...string) RCPIPWhitelistResourceModel {
	return RCPIPWhitelistResourceModel{
		ID:        types.StringUnknown(),
		ContactID: types.Int64Value(7),
		Allow:     allowSet(allow),
		OnDestroy: types.StringValue(onDestroy),
	}
}

func TestRCPIPWhitelistCreateWritesAndReadsBack(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{allow: `["192.0.2.1","2001:db8::/32"]`}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	plan := whitelistState(ctx, t, res, whitelistModel("retain", "2001:db8::/32", "192.0.2.1"))
	resp := &resource.CreateResponse{State: plan}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(plan)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	if len(f.puts) != 1 || f.puts[0] != `{"rcp_client_ip_list":{"allow":["192.0.2.1","2001:db8::/32"]}}` {
		t.Errorf("unexpected PUT bodies %v", f.puts)
	}
	var got RCPIPWhitelistResourceModel
	resp.State.Get(ctx, &got)
	if got.ID.ValueString() != "7" || len(got.Allow.Elements()) != 2 {
		t.Errorf("unexpected state %+v", got)
	}
}

func TestRCPIPWhitelistReadReportsDrift(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{allow: `["192.0.2.1","203.0.113.9"]`}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	prior := whitelistModel("retain", "192.0.2.1")
	prior.ID = types.StringValue("7")
	state := whitelistState(ctx, t, res, prior)
	resp := &resource.ReadResponse{State: state}
	res.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}

	var got RCPIPWhitelistResourceModel
	resp.State.Get(ctx, &got)
	if !got.Allow.Equal(allowSet([]string{"192.0.2.1", "203.0.113.9"})) {
		t.Errorf("drift not reflected in state: %v", got.Allow)
	}
}

func TestRCPIPWhitelistReadGoneContactRemovesResource(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{status: http.StatusNotFound}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	state := whitelistState(ctx, t, res, whitelistModel("retain", "192.0.2.1"))
	resp := &resource.ReadResponse{State: state}
	res.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

func TestRCPIPWhitelistUpdateReplacesList(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{allow: `["198.51.100.0/24"]`}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	prior := whitelistModel("retain", "192.0.2.1")
	prior.ID = types.StringValue("7")
	state := whitelistState(ctx, t, res, prior)
	plan := whitelistState(ctx, t, res, whitelistModel("retain", "198.51.100.0/24"))
	resp := &resource.UpdateResponse{State: state}
	res.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan(plan), State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if len(f.puts) != 1 || f.puts[0] != `{"rcp_client_ip_list":{"allow":["198.51.100.0/24"]}}` {
		t.Errorf("unexpected PUT bodies %v", f.puts)
	}
}

func TestRCPIPWhitelistDeleteRetainMakesNoCall(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{allow: `["192.0.2.1"]`}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	state := whitelistState(ctx, t, res, whitelistModel("retain", "192.0.2.1"))
	resp := &resource.DeleteResponse{State: state}
	res.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() || len(f.puts) != 0 {
		t.Fatalf("expected no API call and no error, got puts=%v diags=%v", f.puts, resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Error("expected a warning that the whitelist stays in force")
	}
}

func TestRCPIPWhitelistDeleteClearEmptiesList(t *testing.T) {
	ctx := context.Background()
	f := &fakeContact{}
	server := httptest.NewServer(f.handler(t))
	defer server.Close()

	res := whitelistResource(server)
	state := whitelistState(ctx, t, res, whitelistModel("clear", "192.0.2.1"))
	resp := &resource.DeleteResponse{State: state}
	res.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete: %v", resp.Diagnostics)
	}
	if len(f.puts) != 1 || f.puts[0] != `{"rcp_client_ip_list":{"allow":[]}}` {
		t.Errorf("unexpected PUT bodies %v", f.puts)
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Error("expected a warning that the whitelist is switched off")
	}
}

func TestRCPIPWhitelistImport(t *testing.T) {
	ctx := context.Background()
	res := &RCPIPWhitelistResource{}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	newState := func() tfsdk.State {
		return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	}

	ok := &resource.ImportStateResponse{State: newState()}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "7"}, ok)
	if ok.Diagnostics.HasError() {
		t.Fatalf("import: %v", ok.Diagnostics)
	}
	var got RCPIPWhitelistResourceModel
	_ = ok.State.Get(ctx, &got)
	if got.ContactID.ValueInt64() != 7 || got.ID.ValueString() != "7" || got.OnDestroy.ValueString() != "retain" {
		t.Errorf("unexpected imported state %+v", got)
	}

	for _, bad := range []string{"abc", "0", "-3", ""} {
		bad := bad
		resp := &resource.ImportStateResponse{State: newState()}
		res.ImportState(ctx, resource.ImportStateRequest{ID: bad}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("expected an error for import id %q", bad)
		}
	}
}

func TestIPListValidator(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		in      []string
		wantErr bool
	}{
		"ipv4":        {[]string{"192.0.2.1"}, false},
		"ipv6":        {[]string{"2001:db8::1"}, false},
		"cidr4":       {[]string{"10.0.0.0/8"}, false},
		"cidr6":       {[]string{"2001:db8::/32"}, false},
		"mixed":       {[]string{"192.0.2.1", "2001:db8::/32"}, false},
		"empty":       {[]string{}, true},
		"hostname":    {[]string{"example.com"}, true},
		"bad octet":   {[]string{"256.0.0.1"}, true},
		"bad prefix":  {[]string{"10.0.0.0/33"}, true},
		"range":       {[]string{"10.0.0.1-10.0.0.9"}, true},
		"zone":        {[]string{"fe80::1%eth0"}, true},
		"blank":       {[]string{""}, true},
		"one bad one": {[]string{"192.0.2.1", "nope"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.SetResponse{}
			ipListValidator{}.ValidateSet(ctx, validator.SetRequest{ConfigValue: allowSet(tc.in)}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestIPListValidatorSkipsUnknownAndNull(t *testing.T) {
	for _, v := range []types.Set{types.SetNull(types.StringType), types.SetUnknown(types.StringType)} {
		resp := &validator.SetResponse{}
		ipListValidator{}.ValidateSet(context.Background(), validator.SetRequest{ConfigValue: v}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected error for %v", v)
		}
	}
}
