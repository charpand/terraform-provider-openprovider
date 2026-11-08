package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ownerAPI serves one domain and records the bodies of the updates to it. An
// accepted update changes the owner that later reads return. A non-empty
// refusal makes each update fail with a non-zero code in a 2xx response.
func ownerAPI(t *testing.T, owner, refusal string, updates *[]map[string]any) *httptest.Server {
	t.Helper()
	domain := func() string {
		return `{"id":1,"status":"ACT","autorenew":"off","owner_handle":"` + owner + `",` +
			`"domain":{"name":"example","extension":"com"}}`
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":{"results":[` + domain() + `],"total":1}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/domains/1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
			}
			*updates = append(*updates, body)
			if refusal != "" {
				_, _ = w.Write([]byte(`{"code":399,"desc":"` + refusal + `"}`))
				return
			}
			if handle, ok := body["owner_handle"].(string); ok {
				owner = handle
			}
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain() + `}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func ownerModel(owner string, autorenew bool) DomainModel {
	return DomainModel{
		ID:             types.StringValue("example.com"),
		Domain:         types.StringValue("example.com"),
		Status:         types.StringValue("ACT"),
		Autorenew:      types.BoolValue(autorenew),
		OwnerHandle:    types.StringValue(owner),
		DnssecKeys:     types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}),
		AdditionalData: types.MapNull(types.StringType),
	}
}

func updateOwner(ctx context.Context, t *testing.T, server *httptest.Server, prior, plan DomainModel) resource.UpdateResponse {
	t.Helper()
	r := &DomainResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test"})}
	priorState := stateOf(ctx, t, r, prior)
	resp := resource.UpdateResponse{State: priorState}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
		State: priorState,
	}, &resp)
	return resp
}

// TestDomainResourceUpdateChangesOwner covers a changed `owner_handle`: the
// update names the new owner, accepts no fee, and the state holds the owner
// that the API returns.
func TestDomainResourceUpdateChangesOwner(t *testing.T) {
	ctx := context.Background()
	var updates []map[string]any
	server := ownerAPI(t, "XX000001-NL", "", &updates)
	defer server.Close()

	resp := updateOwner(ctx, t, server, ownerModel("XX000001-NL", false), ownerModel("XX000002-NL", false))
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if len(updates) != 1 {
		t.Fatalf("got %d updates, want 1", len(updates))
	}
	if got := updates[0]["owner_handle"]; got != "XX000002-NL" {
		t.Errorf("owner_handle in the request: got %v, want XX000002-NL", got)
	}
	if _, ok := updates[0]["accept_update_fee"]; ok {
		t.Errorf("the request accepts a fee: %v", updates[0])
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.OwnerHandle.ValueString() != "XX000002-NL" {
		t.Errorf("owner_handle in the state: got %v, want XX000002-NL", got.OwnerHandle)
	}
}

// TestDomainResourceUpdateKeepsOwnerOutOfOtherUpdates covers an update of a
// different field: it must not name the owner.
func TestDomainResourceUpdateKeepsOwnerOutOfOtherUpdates(t *testing.T) {
	var updates []map[string]any
	server := ownerAPI(t, "XX000001-NL", "", &updates)
	defer server.Close()

	resp := updateOwner(context.Background(), t, server, ownerModel("XX000001-NL", false), ownerModel("XX000001-NL", true))
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if len(updates) != 1 {
		t.Fatalf("got %d updates, want 1", len(updates))
	}
	if _, ok := updates[0]["owner_handle"]; ok {
		t.Errorf("the request names the owner: %v", updates[0])
	}
}

// TestDomainResourceUpdateReportsARefusedOwnerChange covers a registry that
// requires a fee or a trade: the apply fails with the reason of the registrar,
// and the state keeps the old owner.
func TestDomainResourceUpdateReportsARefusedOwnerChange(t *testing.T) {
	ctx := context.Background()
	var updates []map[string]any
	server := ownerAPI(t, "XX000001-NL", "This change requires a trade", &updates)
	defer server.Close()

	resp := updateOwner(ctx, t, server, ownerModel("XX000001-NL", false), ownerModel("XX000002-NL", false))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for a refused change of owner")
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "requires a trade") {
		t.Errorf("the error does not state the reason: %s", detail)
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.OwnerHandle.ValueString() != "XX000001-NL" {
		t.Errorf("owner_handle in the state: got %v, want XX000001-NL", got.OwnerHandle)
	}
}
