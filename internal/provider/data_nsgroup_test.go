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

// readNSGroup reads the group `missing-or-held` from an API that answers
// with `status` and `body`.
func readNSGroup(t *testing.T, status int, body string, allowMissing types.Bool) (NSGroupModel, *datasource.ReadResponse) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	d := &NSGroupDataSource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, NSGroupModel{Name: types.StringValue("missing-or-held"), AllowMissing: allowMissing}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	var state NSGroupModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, resp
}

const heldGroup = `{"code":0,"data":{"name":"missing-or-held","name_servers":[{"name":"ns1.example.com"}]}}`

func TestNSGroupMissingIsAnErrorByDefault(t *testing.T) {
	for name, allowMissing := range map[string]types.Bool{"unset": types.BoolNull(), "false": types.BoolValue(false)} {
		t.Run(name, func(t *testing.T) {
			if _, resp := readNSGroup(t, http.StatusNotFound, `{}`, allowMissing); !resp.Diagnostics.HasError() {
				t.Fatal("a missing group should be an error without `allow_missing`")
			}
		})
	}
}

func TestNSGroupMissingIsAResultWhenAllowed(t *testing.T) {
	state, resp := readNSGroup(t, http.StatusNotFound, `{}`, types.BoolValue(true))
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if state.Exists.ValueBool() || state.Exists.IsNull() {
		t.Errorf("exists: got %v, want false", state.Exists)
	}
	if state.ID.ValueString() != "missing-or-held" || state.Name.ValueString() != "missing-or-held" {
		t.Errorf("id and name: got %v and %v", state.ID, state.Name)
	}
	if len(state.Nameservers) != 0 {
		t.Errorf("nameservers: got %v, want none", state.Nameservers)
	}
}

func TestNSGroupHeldExists(t *testing.T) {
	for name, allowMissing := range map[string]types.Bool{"unset": types.BoolNull(), "true": types.BoolValue(true)} {
		t.Run(name, func(t *testing.T) {
			state, resp := readNSGroup(t, http.StatusOK, heldGroup, allowMissing)
			if resp.Diagnostics.HasError() {
				t.Fatalf("read: %v", resp.Diagnostics)
			}
			if !state.Exists.ValueBool() {
				t.Errorf("exists: got %v, want true", state.Exists)
			}
			if len(state.Nameservers) != 1 {
				t.Errorf("nameservers: got %v, want one", state.Nameservers)
			}
			if !state.AllowMissing.Equal(allowMissing) {
				t.Errorf("allow_missing: got %v, want the configured %v", state.AllowMissing, allowMissing)
			}
		})
	}
}

// TestNSGroupFailedReadIsAnError guards against a failed read that reads as
// a missing group: a configuration would then create a group that exists.
func TestNSGroupFailedReadIsAnError(t *testing.T) {
	if _, resp := readNSGroup(t, http.StatusInternalServerError, `{}`, types.BoolValue(true)); !resp.Diagnostics.HasError() {
		t.Fatal("a failed read should be an error, also with `allow_missing`")
	}
}
