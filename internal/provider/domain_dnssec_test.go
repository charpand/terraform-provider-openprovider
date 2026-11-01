package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// noDnssecKeys is the value of `dnssec_keys = []`.
func noDnssecKeys() types.List {
	return types.ListValueMust(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}, []attr.Value{})
}

// TestDomainResourceCreateTransferWithNoKeysSendsAnEmptyList covers a
// transfer with `dnssec_keys = []`. The transfer request sends an empty list,
// so that the domain does not keep the keys of the registrar it comes from.
// The domain reads back without keys, and the state keeps the empty list, or
// the framework rejects the apply as inconsistent with the plan.
func TestDomainResourceCreateTransferWithNoKeysSendsAnEmptyList(t *testing.T) {
	for _, status := range []string{domainStatusRequested, "ACT"} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			api := newOrderAPI(t, "example", "com", status)
			defer api.Close()
			r := &DomainResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})}

			got := createDomain(ctx, t, r, DomainModel{
				ID:              types.StringUnknown(),
				Domain:          types.StringValue("example.com"),
				OwnerHandle:     types.StringValue("XX000001-NL"),
				AuthCode:        types.StringValue("secret"),
				Autorenew:       types.BoolValue(false),
				Period:          types.Int64Unknown(),
				Status:          types.StringUnknown(),
				DnssecKeys:      noDnssecKeys(),
				IsDnssecEnabled: types.BoolUnknown(),
				OnDestroy:       types.StringValue(onDestroyRetain),
			})

			if api.ordered != "/v1beta/domains/transfer" {
				t.Fatalf("order sent to %q, want the transfer", api.ordered)
			}
			if sent, ok := api.order["dnssec_keys"].([]any); !ok || len(sent) != 0 {
				t.Errorf("the transfer should send an empty list, sent %v", api.order["dnssec_keys"])
			}
			if !got.DnssecKeys.Equal(noDnssecKeys()) {
				t.Errorf("dnssec_keys: got %v, want an empty list", got.DnssecKeys)
			}
		})
	}
}

// TestDomainResourceUpdateWithNoKeysRemovesTheKeys covers an update from a
// domain with keys to `dnssec_keys = []`. The update request sends an empty
// list, which removes the keys. The domain reads back without keys, and the
// state keeps the empty list, so that the next plan has no difference.
func TestDomainResourceUpdateWithNoKeysRemovesTheKeys(t *testing.T) {
	ctx := context.Background()
	domain := `{"id":1,"status":"ACT","autorenew":"off","owner_handle":"XX000001-NL",` +
		`"domain":{"name":"example","extension":"com"}}`
	var sent map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":{"results":[` + domain + `],"total":1}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1beta/domains/1":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read update: %v", err)
			}
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Errorf("decode update %s: %v", body, err)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain + `}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	r := &DomainResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test"})}

	prior := DomainModel{
		ID:              types.StringValue("example.com"),
		Domain:          types.StringValue("example.com"),
		Status:          types.StringValue("ACT"),
		Autorenew:       types.BoolValue(false),
		OwnerHandle:     types.StringValue("XX000001-NL"),
		DnssecKeys:      dnssecKeys(t, "AwEAAa"),
		IsDnssecEnabled: types.BoolValue(true),
	}
	plan := prior
	plan.DnssecKeys = noDnssecKeys()
	plan.IsDnssecEnabled = types.BoolUnknown()

	priorState := stateOf(ctx, t, r, prior)
	resp := resource.UpdateResponse{State: priorState}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
		State: priorState,
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}

	if keys, ok := sent["dnssec_keys"].([]any); !ok || len(keys) != 0 {
		t.Errorf("the update should send an empty list, sent %v", sent["dnssec_keys"])
	}
	if _, ok := sent["is_dnssec_enabled"]; ok {
		t.Errorf("the update should not send an unknown is_dnssec_enabled, sent %v", sent["is_dnssec_enabled"])
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if !got.DnssecKeys.Equal(noDnssecKeys()) {
		t.Errorf("dnssec_keys: got %v, want an empty list", got.DnssecKeys)
	}
	if !got.IsDnssecEnabled.Equal(types.BoolValue(false)) {
		t.Errorf("is_dnssec_enabled: got %v, want false", got.IsDnssecEnabled)
	}
}

// TestDomainResourceReadKeepsAnEmptyList covers a refresh of a domain with
// no keys. A prior empty list stays an empty list, and a prior null stays
// null.
func TestDomainResourceReadKeepsAnEmptyList(t *testing.T) {
	for name, prior := range map[string]types.List{
		"empty list": noDnssecKeys(),
		"null":       types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}),
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			server := listingOf(t, "example", "com")
			defer server.Close()
			r := &DomainResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test"})}

			state := stateOf(ctx, t, r, DomainModel{
				ID:          types.StringValue("example.com"),
				Domain:      types.StringValue("example.com"),
				OwnerHandle: types.StringValue("XX000001-NL"),
				Status:      types.StringValue("ACT"),
				DnssecKeys:  prior,
			})
			resp := &resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("read: %v", resp.Diagnostics)
			}
			var got DomainModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("get: %v", diags)
			}
			if !got.DnssecKeys.Equal(prior) {
				t.Errorf("dnssec_keys: got %v, want %v", got.DnssecKeys, prior)
			}
		})
	}
}
