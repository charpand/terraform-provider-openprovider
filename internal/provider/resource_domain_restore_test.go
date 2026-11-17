package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// restoreAPI stands in for an account with example.com as the id 1. `listed`
// puts the domain in the normal listing, `deleted` in the listing of deleted
// domains. With neither the account has no such domain.
type restoreAPI struct {
	*httptest.Server
	listed  bool
	deleted bool
	refuse  bool
	posts   []map[string]any
}

func newRestoreAPI(t *testing.T, listed, deleted bool) *restoreAPI {
	t.Helper()
	api := &restoreAPI{listed: listed, deleted: deleted}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		const entry = `{"code":0,"data":{"results":[{"id":1,"status":%q,"domain":{"name":"example","extension":"com"}}],"total":1}}`
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains":
			switch {
			case r.URL.Query().Get("is_deleted") == "true" && api.deleted:
				_, _ = fmt.Fprintf(w, entry, "DEL")
			case r.URL.Query().Get("is_deleted") != "true" && api.listed:
				_, _ = fmt.Fprintf(w, entry, "ACT")
			default:
				_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[],"total":0}}`)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v1/domains/1/restore":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode restore: %v", err)
			}
			api.posts = append(api.posts, body)
			if api.refuse {
				_, _ = fmt.Fprint(w, `{"code":399,"desc":"The domain cannot be restored"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"status":"PEN"}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func createRestore(t *testing.T, api *restoreAPI) *resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	r := &DomainRestoreResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	plan := DomainRestoreModel{
		ID:       types.StringUnknown(),
		Domain:   types.StringValue("example.com"),
		Triggers: types.MapNull(types.StringType),
		Status:   types.StringUnknown(),
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	return resp
}

func TestDomainRestoreSendsTheRestore(t *testing.T) {
	for name, api := range map[string]*restoreAPI{
		"a listed domain":  newRestoreAPI(t, true, false),
		"a deleted domain": newRestoreAPI(t, false, true),
	} {
		t.Run(name, func(t *testing.T) {
			resp := createRestore(t, api)
			if resp.Diagnostics.HasError() {
				t.Fatalf("create: %v", resp.Diagnostics)
			}
			want := []map[string]any{{"id": float64(1), "domain": map[string]any{"name": "example", "extension": "com"}}}
			if !reflect.DeepEqual(api.posts, want) {
				t.Errorf("posts: got %v, want %v", api.posts, want)
			}
			var got DomainRestoreModel
			if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
				t.Fatalf("get: %v", diags)
			}
			if got.ID.ValueString() != "example.com" || got.Status.ValueString() != "PEN" {
				t.Errorf("state: got %+v", got)
			}
		})
	}
}

func TestDomainRestoreWithoutTheDomain(t *testing.T) {
	api := newRestoreAPI(t, false, false)
	resp := createRestore(t, api)
	if !resp.Diagnostics.HasError() {
		t.Fatal("the create should fail")
	}
	if len(api.posts) != 0 {
		t.Errorf("the provider should send no restore: %v", api.posts)
	}
}

func TestDomainRestoreRefused(t *testing.T) {
	api := newRestoreAPI(t, true, false)
	api.refuse = true
	resp := createRestore(t, api)
	if !resp.Diagnostics.HasError() {
		t.Fatal("the create should fail")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("the create should keep no state")
	}
}
