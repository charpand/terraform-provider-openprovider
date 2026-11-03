package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// checkAPI stands in for the registry's availability check: it answers every
// check with the given status and premium flag, and records the body it got.
func checkAPI(t *testing.T, status string, premium bool, bodies *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/v1/domains/check" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		*bodies = append(*bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"results": []map[string]any{{
					"domain":     "example.com",
					"status":     status,
					"is_premium": premium,
				}},
			},
		})
	}))
}

func checkDomain(t *testing.T, status string, premium bool) (DomainCheckModel, []map[string]any, bool) {
	t.Helper()
	var bodies []map[string]any
	server := checkAPI(t, status, premium, &bodies)
	defer server.Close()
	d := &DomainCheckDataSource{client: client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})}
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	// A config is set through a state of the same schema: `tfsdk.Config`
	// has no setter of its own.
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, DomainCheckModel{
		Domain:    types.StringValue("example.com"),
		ID:        types.StringUnknown(),
		Status:    types.StringUnknown(),
		Available: types.BoolUnknown(),
		IsPremium: types.BoolUnknown(),
		Reason:    types.StringUnknown(),
	}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	var state DomainCheckModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, bodies, resp.Diagnostics.HasError()
}

func TestDomainCheckFree(t *testing.T) {
	state, bodies, errored := checkDomain(t, "free", false)
	if errored {
		t.Fatal("a free domain should not error")
	}
	if !state.Available.ValueBool() || state.Status.ValueString() != "free" || state.IsPremium.ValueBool() {
		t.Fatalf("a free domain should be available and not premium, got %+v", state)
	}
	if len(bodies) != 1 {
		t.Fatalf("expected one check, got %d", len(bodies))
	}
	domains, _ := bodies[0]["domains"].([]any)
	if len(domains) != 1 {
		t.Fatalf("expected one domain in the check, got %v", bodies[0])
	}
	checked, _ := domains[0].(map[string]any)
	if checked["name"] != "example" || checked["extension"] != "com" {
		t.Fatalf("expected example.com split into name and extension, got %v", checked)
	}
}

func TestDomainCheckTaken(t *testing.T) {
	state, _, errored := checkDomain(t, "active", false)
	if errored {
		t.Fatal("a taken domain is a result, not an error")
	}
	if state.Available.ValueBool() || state.Status.ValueString() != "active" {
		t.Fatalf("a taken domain should not be available, got %+v", state)
	}
}

func TestDomainCheckPremium(t *testing.T) {
	state, _, errored := checkDomain(t, "free", true)
	if errored {
		t.Fatal("a premium domain should not error")
	}
	if !state.Available.ValueBool() || !state.IsPremium.ValueBool() {
		t.Fatalf("a free premium domain should be available and premium, got %+v", state)
	}
}
