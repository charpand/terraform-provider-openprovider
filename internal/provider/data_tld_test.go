package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func readTLD(t *testing.T, status int, body string) (TLDModel, *datasource.ReadResponse, string) {
	t.Helper()
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	d := &TLDDataSource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, TLDModel{Name: types.StringValue("cfd")}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	var state TLDModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, resp, path
}

func TestTLDReadsTheRoles(t *testing.T) {
	state, resp, path := readTLD(t, http.StatusOK, `{"code":0,"data":{"name":"cfd","owner_handle_supported":true,"admin_handle_supported":false,"tech_handle_supported":true,"billing_handle_supported":false,"reseller_handle_supported":false}}`)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if path != "/v1/tlds/cfd" {
		t.Errorf("path: got %s", path)
	}
	if state.ID.ValueString() != "cfd" {
		t.Errorf("id: got %v", state.ID)
	}
	for name, tc := range map[string]struct {
		got  types.Bool
		want bool
	}{
		"owner":    {state.OwnerHandleSupported, true},
		"admin":    {state.AdminHandleSupported, false},
		"tech":     {state.TechHandleSupported, true},
		"billing":  {state.BillingHandleSupported, false},
		"reseller": {state.ResellerHandleSupported, false},
	} {
		if tc.got.IsNull() || tc.got.ValueBool() != tc.want {
			t.Errorf("%s: got %v, want %v", name, tc.got, tc.want)
		}
	}
}

// TestTLDUnreportedRoleIsNull guards against a role that the API leaves out
// and that then reads as a role that the extension does not take.
func TestTLDUnreportedRoleIsNull(t *testing.T) {
	state, resp, _ := readTLD(t, http.StatusOK, `{"code":0,"data":{"name":"cfd","tech_handle_supported":true}}`)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if !state.AdminHandleSupported.IsNull() {
		t.Errorf("admin: got %v, want null", state.AdminHandleSupported)
	}
	if !state.TechHandleSupported.ValueBool() {
		t.Errorf("tech: got %v, want true", state.TechHandleSupported)
	}
}

func TestTLDFailedReadIsAnError(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"unknown extension": {http.StatusNotFound, `{}`},
		"server error":      {http.StatusInternalServerError, `{}`},
		"no name":           {http.StatusOK, `{"code":0,"data":{}}`},
	} {
		t.Run(name, func(t *testing.T) {
			if _, resp, _ := readTLD(t, tc.status, tc.body); !resp.Diagnostics.HasError() {
				t.Fatal("a failed read should be an error")
			}
		})
	}
}
