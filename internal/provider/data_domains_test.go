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

// listingAPI stands in for an account holding example.com and example.org,
// filtering on `full_name` as the API does, and records each query it got.
func listingAPI(t *testing.T, queries *[]string) *httptest.Server {
	t.Helper()
	held := []map[string]any{
		{"id": 123, "domain": map[string]string{"name": "example", "extension": "com"}, "status": "ACT", "autorenew": "on"},
		{"id": 456, "domain": map[string]string{"name": "example", "extension": "org"}, "status": "ACT", "autorenew": "off"},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/domains" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		*queries = append(*queries, r.URL.RawQuery)
		results := []map[string]any{}
		for _, domain := range held {
			parts, _ := domain["domain"].(map[string]string)
			name := parts["name"] + "." + parts["extension"]
			if want := r.URL.Query().Get("full_name"); want == "" || want == name {
				results = append(results, domain)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"results": results, "total": len(results)},
		})
	}))
}

func listDomains(t *testing.T, fullName types.String) (DomainsModel, []string, bool) {
	t.Helper()
	var queries []string
	server := listingAPI(t, &queries)
	defer server.Close()
	d := &DomainsDataSource{client: client.NewClient(client.Config{
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
	if diags := staged.Set(ctx, DomainsModel{
		ID:       types.StringNull(),
		FullName: fullName,
	}); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	var state DomainsModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, queries, resp.Diagnostics.HasError()
}

func TestDomainsListsAll(t *testing.T) {
	state, queries, errored := listDomains(t, types.StringNull())
	if errored {
		t.Fatal("listing should not error")
	}
	if len(state.Domains) != 2 || state.ID.ValueString() != "all" {
		t.Fatalf("expected both domains under id all, got %+v", state)
	}
	if len(queries) != 1 || queries[0] != "limit=100" {
		t.Fatalf("an unfiltered listing should page with an explicit limit and no other filter, sent %v", queries)
	}
	if !state.Domains[0].Autorenew.ValueBool() || state.Domains[1].Autorenew.ValueBool() {
		t.Fatalf("autorenew should follow the API's on/off, got %+v", state.Domains)
	}
}

func TestDomainsFiltersByFullName(t *testing.T) {
	state, queries, errored := listDomains(t, types.StringValue("example.org"))
	if errored {
		t.Fatal("a filtered listing should not error")
	}
	if len(state.Domains) != 1 || state.Domains[0].Domain.ValueString() != "example.org" || state.Domains[0].ID.ValueInt64() != 456 {
		t.Fatalf("expected example.org alone, got %+v", state.Domains)
	}
	if len(queries) != 1 || queries[0] != "full_name=example.org&limit=100" {
		t.Fatalf("a filtered listing should filter on the API's side, sent %v", queries)
	}
}

func TestDomainsEmptyWhenNotHeld(t *testing.T) {
	state, _, errored := listDomains(t, types.StringValue("example.net"))
	if errored {
		t.Fatal("a domain the account does not hold is an empty list, not an error")
	}
	if len(state.Domains) != 0 || state.ID.ValueString() != "example.net" {
		t.Fatalf("expected an empty list under the filter's id, got %+v", state)
	}
}
