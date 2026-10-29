package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// readAPI stands in for OpenProvider with an account whose listing answers
// as `listing` says, and a registry whose availability check answers
// `checkStatus` for every name.
func readAPI(t *testing.T, listing map[string]any, checkStatus string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_ = json.NewEncoder(w).Encode(listing)
		case r.Method == http.MethodPost && r.URL.Path == "/v1beta/domains/check":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"results": []map[string]any{{"domain": "example.com", "status": checkStatus}},
				},
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

var emptyListing = map[string]any{
	"code": 0,
	"data": map[string]any{"results": []map[string]any{}, "total": 0},
}

// heldDomainState is the state of a domain the account registered earlier.
func heldDomainState(t *testing.T, r *DomainResource) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(ctx, DomainModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		OwnerHandle: types.StringValue("XX000000-NL"),
		DnssecKeys:  types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}),
	})
	if diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}
	return state
}

func readDomain(t *testing.T, listing map[string]any, checkStatus string) *resource.ReadResponse {
	t.Helper()
	server := readAPI(t, listing, checkStatus)
	defer server.Close()
	r := &DomainResource{client: client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})}
	state := heldDomainState(t, r)
	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
	return resp
}

func TestDomainResourceReadDropsAFreeName(t *testing.T) {
	resp := readDomain(t, emptyListing, "free")
	if resp.Diagnostics.HasError() {
		t.Fatalf("a name gone from the account should read without error, got %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("a name the registry calls free should leave the state")
	}
}

func TestDomainResourceReadKeepsATakenName(t *testing.T) {
	resp := readDomain(t, emptyListing, "active")
	if resp.Diagnostics.HasError() {
		t.Fatalf("a name missing from a listing should read without error, got %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a name the registry calls taken should stay in state when the listing misses it")
	}
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Fatal("keeping a name the listing misses should warn")
	}
}

func TestDomainResourceReadRefusesARefusedListing(t *testing.T) {
	resp := readDomain(t, map[string]any{
		"code": 196,
		"desc": "Authentication failure",
		"data": map[string]any{"results": []map[string]any{}, "total": 0},
	}, "free")
	if !resp.Diagnostics.HasError() {
		t.Fatal("a listing the API refused should be an error, not an empty account")
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a listing the API refused should leave the state alone")
	}
}

// TestDomainResourceReadKeepsTheDnssecOfATransferInFlight covers a refresh
// after a transfer with DNSSEC keys. While the status is `REQ`, the registry
// does not have the keys, and the state keeps them. When the transfer
// completes, the state holds what the registry has, so a difference from the
// configuration plans an update that sends the keys.
func TestDomainResourceReadKeepsTheDnssecOfATransferInFlight(t *testing.T) {
	ctx := context.Background()
	keys := dnssecKeys(t, "AwEAAa")

	for status, want := range map[string]struct {
		keys    types.List
		enabled types.Bool
	}{
		domainStatusRequested: {keys, types.BoolValue(true)},
		domainStatusPending:   {keys, types.BoolValue(true)},
		domainStatusActive:    {types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}), types.BoolValue(false)},
	} {
		t.Run(status, func(t *testing.T) {
			server := readAPI(t, map[string]any{
				"code": 0,
				"data": map[string]any{
					"results": []map[string]any{{
						"id":                1,
						"domain":            map[string]string{"name": "example", "extension": "com"},
						"status":            status,
						"is_dnssec_enabled": false,
					}},
					"total": 1,
				},
			}, "")
			defer server.Close()
			r := &DomainResource{client: client.NewClient(client.Config{
				BaseURL:    server.URL,
				Token:      "test",
				HTTPClient: server.Client(),
			})}
			state := stateOf(ctx, t, r, DomainModel{
				ID:              types.StringValue("example.com"),
				Domain:          types.StringValue("example.com"),
				OwnerHandle:     types.StringValue("XX000000-NL"),
				Status:          types.StringValue(domainStatusRequested),
				DnssecKeys:      keys,
				IsDnssecEnabled: types.BoolValue(true),
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
			if !got.DnssecKeys.Equal(want.keys) {
				t.Errorf("dnssec_keys: got %v, want %v", got.DnssecKeys, want.keys)
			}
			if !got.IsDnssecEnabled.Equal(want.enabled) {
				t.Errorf("is_dnssec_enabled: got %v, want %v", got.IsDnssecEnabled, want.enabled)
			}
		})
	}
}
