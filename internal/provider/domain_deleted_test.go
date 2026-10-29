package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// deletedAPI stands in for OpenProvider after example.com left the account,
// for example with a transfer to a different registrar. The default listing
// answers `listed`, the listing with `is_deleted=true` answers `deleted`, and
// the availability check answers `checkStatus`. It records the `is_deleted`
// filter of each listing and the body of each update, and fails the test on
// any delete.
type deletedAPI struct {
	*httptest.Server
	filters []string
	updates []map[string]any
}

func newDeletedAPI(t *testing.T, listed, deleted []map[string]any, checkStatus string) *deletedAPI {
	t.Helper()
	api := &deletedAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			filter := r.URL.Query().Get("is_deleted")
			api.filters = append(api.filters, filter)
			results := listed
			if filter == "true" {
				results = deleted
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"results": results, "total": len(results)},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1beta/domains/check" && checkStatus != "":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"results": []map[string]any{
					{"domain": "example.com", "status": checkStatus},
				}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/v1beta/domains/123":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding update: %v", err)
			}
			api.updates = append(api.updates, body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"id": 123},
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return api
}

// listedExample is example.com in a listing, with its autorenew on.
func listedExample(id int, status string) map[string]any {
	return map[string]any{
		"id":        id,
		"domain":    map[string]string{"name": "example", "extension": "com"},
		"status":    status,
		"autorenew": "on",
	}
}

func (api *deletedAPI) resource() *DomainResource {
	return &DomainResource{client: client.NewClient(client.Config{
		BaseURL:    api.URL,
		Token:      "test",
		HTTPClient: api.Client(),
	})}
}

func (api *deletedAPI) delete(t *testing.T, onDestroy string) *resource.DeleteResponse {
	t.Helper()
	r := api.resource()
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: domainState(t, r, types.StringValue(onDestroy))}, resp)
	return resp
}

func (api *deletedAPI) read(t *testing.T) *resource.ReadResponse {
	t.Helper()
	r := api.resource()
	state := heldDomainState(t, r)
	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
	return resp
}

func TestDomainResourceDeleteLeavesADeletedDomain(t *testing.T) {
	for _, onDestroy := range []string{onDestroyExpire, onDestroyDelete} {
		t.Run(onDestroy, func(t *testing.T) {
			api := newDeletedAPI(t, []map[string]any{listedExample(123, domainStatusDeleted)}, nil, "")
			defer api.Close()

			resp := api.delete(t, onDestroy)
			if resp.Diagnostics.HasError() {
				t.Fatalf("a domain the account shows as deleted should destroy without error, got %v", resp.Diagnostics)
			}
			if resp.Diagnostics.WarningsCount() == 0 {
				t.Error("a domain the account shows as deleted should warn")
			}
			if len(api.updates) != 0 {
				t.Errorf("a domain the account shows as deleted should send no update, sent %v", api.updates)
			}
		})
	}
}

func TestDomainResourceDeleteExpiresWhenOnlyTheDeletedListingShowsIt(t *testing.T) {
	api := newDeletedAPI(t, nil, []map[string]any{listedExample(123, domainStatusDeleted)}, "active")
	defer api.Close()

	resp := api.delete(t, onDestroyExpire)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a domain only the deleted listing shows should destroy without error, got %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Error("a domain only the deleted listing shows should warn")
	}
	if len(api.updates) != 0 {
		t.Errorf("a domain only the deleted listing shows should send no update, sent %v", api.updates)
	}
	if len(api.filters) != 2 || api.filters[0] != "" || api.filters[1] != "true" {
		t.Errorf("listings: got filters %q, want the default listing and then is_deleted=true", api.filters)
	}
}

func TestDomainResourceDeleteExpiresErrorsWhenNeitherListingShowsIt(t *testing.T) {
	api := newDeletedAPI(t, nil, nil, "active")
	defer api.Close()

	resp := api.delete(t, onDestroyExpire)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a domain that is taken and that neither listing shows should error, as its autorenew may still be on")
	}
	if len(api.updates) != 0 {
		t.Errorf("a domain that is not listed should send no update, sent %v", api.updates)
	}
	if len(api.filters) != 2 || api.filters[1] != "true" {
		t.Errorf("listings: got filters %q, want the deleted listing asked too", api.filters)
	}
}

func TestDomainResourceDeleteExpiresTheHeldDomainNextToADeletedOne(t *testing.T) {
	api := newDeletedAPI(t, []map[string]any{
		listedExample(99, domainStatusDeleted),
		listedExample(123, domainStatusActive),
	}, nil, "")
	defer api.Close()

	resp := api.delete(t, onDestroyExpire)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expiring a domain should not error, got %v", resp.Diagnostics)
	}
	if len(api.updates) != 1 || api.updates[0]["autorenew"] != "off" {
		t.Fatalf("expiring the held domain should turn its autorenew off, sent %v", api.updates)
	}
}

func TestDomainResourceReadDropsADeletedDomain(t *testing.T) {
	for name, api := range map[string]*deletedAPI{
		"listed as deleted":           newDeletedAPI(t, []map[string]any{listedExample(123, domainStatusDeleted)}, nil, ""),
		"only in the deleted listing": newDeletedAPI(t, nil, []map[string]any{listedExample(123, domainStatusDeleted)}, "active"),
	} {
		t.Run(name, func(t *testing.T) {
			defer api.Close()
			resp := api.read(t)
			if resp.Diagnostics.HasError() {
				t.Fatalf("a deleted domain should read without error, got %v", resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Fatal("a domain the account shows only as deleted should leave the state")
			}
		})
	}
}

func TestDomainResourceReadKeepsATakenNameNeitherListingShows(t *testing.T) {
	api := newDeletedAPI(t, nil, nil, "active")
	defer api.Close()

	resp := api.read(t)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a name missing from both listings should read without error, got %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a name the registry calls taken should stay in state when neither listing shows it")
	}
}
