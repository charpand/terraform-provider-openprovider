package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// listingOf serves a domain listing that holds one active domain, and
// accepts any update to it. It is all `Update` asks of the API.
func listingOf(t *testing.T, name, extension string) *httptest.Server {
	t.Helper()
	domain := `{"id":1,"status":"ACT","autorenew":"off","owner_handle":"XX000001-NL",` +
		`"domain":{"name":"` + name + `","extension":"` + extension + `"}}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":{"results":[` + domain + `],"total":1}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1beta/domains/1":
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain + `}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// stateOf builds a state or plan of the domain resource's schema from a
// model, the way the framework hands one to `Update`.
func stateOf(ctx context.Context, t *testing.T, r *DomainResource, m DomainModel) tfsdk.State {
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

// TestDomainResourceUpdateKeepsOrderFields covers an update that follows an
// import. The prior state then holds no `period`, `max_cost` or `currency`,
// the API has no record of them, and the plan states all three. The applied
// state must carry the plan's values whether or not the update had a request
// to send, or the framework rejects the result as inconsistent with the plan.
func TestDomainResourceUpdateKeepsOrderFields(t *testing.T) {
	ctx := context.Background()
	server := listingOf(t, "example", "com")
	defer server.Close()
	r := &DomainResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test"})}

	noKeys := types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes})
	prior := DomainModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		Status:      types.StringValue("ACT"),
		Autorenew:   types.BoolValue(false),
		OwnerHandle: types.StringValue("XX000001-NL"),
		DnssecKeys:  noKeys,
	}
	ordered := prior
	ordered.Period = types.Int64Value(1)
	ordered.MaxCost = types.Int64Value(1500)
	ordered.Currency = types.StringValue("EUR")

	for name, autorenew := range map[string]bool{
		"nothing to send": false,
		"a request sent":  true,
	} {
		t.Run(name, func(t *testing.T) {
			plan := ordered
			plan.Autorenew = types.BoolValue(autorenew)
			priorState := stateOf(ctx, t, r, prior)
			resp := resource.UpdateResponse{State: priorState}
			r.Update(ctx, resource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
				State: priorState,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("update: %v", resp.Diagnostics)
			}
			var got DomainModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("get: %v", diags)
			}
			for field, want := range map[string]struct{ got, want attr.Value }{
				"period":   {got.Period, plan.Period},
				"max_cost": {got.MaxCost, plan.MaxCost},
				"currency": {got.Currency, plan.Currency},
			} {
				if !want.got.Equal(want.want) {
					t.Errorf("%s: got %v, want %v", field, want.got, want.want)
				}
			}
		})
	}
}

// TestDomainResourceUpdateResolvesAnUnknownPeriod covers a domain whose
// prior state holds `period` as null -- one that predates the field, or was
// imported before it was ever set -- with nothing else changed. `period`'s
// `UseStateForUnknown` plan modifier can't resolve against a null prior
// state, so the plan handed to `Update` still carries it unknown, the same
// way Terraform's own plan step would produce it. `Update` must not write
// that unknown into the applied state; it should fall back to the prior
// state's (null) value instead.
func TestDomainResourceUpdateResolvesAnUnknownPeriod(t *testing.T) {
	ctx := context.Background()
	server := listingOf(t, "example", "com")
	defer server.Close()
	r := &DomainResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test"})}

	noKeys := types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes})
	prior := DomainModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		Status:      types.StringValue("ACT"),
		Autorenew:   types.BoolValue(false),
		OwnerHandle: types.StringValue("XX000001-NL"),
		DnssecKeys:  noKeys,
	}
	plan := prior
	plan.Period = types.Int64Unknown()

	priorState := stateOf(ctx, t, r, prior)
	resp := resource.UpdateResponse{State: priorState}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
		State: priorState,
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}

	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.Period.IsUnknown() {
		t.Error("period should not be unknown after apply")
	}
	if !got.Period.Equal(prior.Period) {
		t.Errorf("period: got %v, want %v", got.Period, prior.Period)
	}
}
