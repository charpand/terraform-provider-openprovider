package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const heldCustomer = `{"code":0,"data":{"handle":"XX123456-XX","email":"holder@example.com","name":{"last_name":"Holder"}}}`

func readCustomerExists(t *testing.T, status int, body string) (CustomerExistsModel, *datasource.ReadResponse) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	d := &CustomerExistsDataSource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, CustomerExistsModel{Handle: types.StringValue("XX123456-XX")}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	var state CustomerExistsModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, resp
}

func TestCustomerExistsAnswers(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
		want   bool
	}{
		"held":    {http.StatusOK, heldCustomer, true},
		"missing": {http.StatusNotFound, `{}`, false},
		"deleted": {http.StatusOK, `{"code":0,"data":{"handle":"XX123456-XX","is_deleted":true}}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			state, resp := readCustomerExists(t, tc.status, tc.body)
			if resp.Diagnostics.HasError() {
				t.Fatalf("read: %v", resp.Diagnostics)
			}
			if state.Exists.IsNull() || state.Exists.ValueBool() != tc.want {
				t.Errorf("exists: got %v, want %v", state.Exists, tc.want)
			}
			if state.ID.ValueString() != "XX123456-XX" {
				t.Errorf("id: got %v", state.ID)
			}
		})
	}
}

// TestCustomerExistsFailedReadIsAnError guards against an answer that reads
// as a result although the API did not give one.
func TestCustomerExistsFailedReadIsAnError(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"server error": {http.StatusInternalServerError, `{}`},
		"no handle":    {http.StatusOK, `{"code":0,"data":{}}`},
		"not json":     {http.StatusOK, `<html>`},
	} {
		t.Run(name, func(t *testing.T) {
			if _, resp := readCustomerExists(t, tc.status, tc.body); !resp.Diagnostics.HasError() {
				t.Fatal("a failed read should be an error")
			}
		})
	}
}

func TestCustomerExistsKeepsTheCustomerOutOfTheState(t *testing.T) {
	_, resp := readCustomerExists(t, http.StatusOK, heldCustomer)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if got := len(resp.State.Schema.GetAttributes()); got != 3 {
		t.Errorf("the state should hold `id`, `handle` and `exists` only, it has %d attributes", got)
	}
	if strings.Contains(resp.State.Raw.String(), "holder@example.com") {
		t.Error("the state holds the mail address of the customer")
	}
}
