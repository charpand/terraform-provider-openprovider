package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type zoneAPI struct {
	*httptest.Server
	posts   []map[string]any
	deletes []string
}

func newZoneAPI(t *testing.T) *zoneAPI {
	t.Helper()
	api := &zoneAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/dns/zones":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
			}
			api.posts = append(api.posts, body)
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/dns/zones/example.com":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"name":"example","extension":"com","type":"master","provider":"openprovider",`+
				`"creation_date":"2026-01-01 00:00:00","modification_date":"2026-01-02 00:00:00"}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/dns/zones/example.com":
			api.deletes = append(api.deletes, r.URL.Path)
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func zoneResource(api *zoneAPI) *DNSZoneResource {
	return &DNSZoneResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
}

func zoneModel(zoneName string, allowDeletion bool) DNSZoneResourceModel {
	return DNSZoneResourceModel{
		ZoneName:         types.StringValue(zoneName),
		Type:             types.StringValue("master"),
		Provider:         types.StringUnknown(),
		Secured:          types.BoolNull(),
		MasterIP:         types.StringNull(),
		TemplateName:     types.StringNull(),
		AllowDeletion:    types.BoolValue(allowDeletion),
		Name:             types.StringUnknown(),
		Extension:        types.StringUnknown(),
		CreationDate:     types.StringUnknown(),
		ModificationDate: types.StringUnknown(),
		ID:               types.StringUnknown(),
	}
}

func TestDNSZoneCreate(t *testing.T) {
	ctx := context.Background()
	api := newZoneAPI(t)
	r := zoneResource(api)
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := sr.Schema

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, zoneModel("example.com", false))}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	if len(api.posts) != 1 {
		t.Fatalf("posts: got %d, want 1", len(api.posts))
	}
	domain, _ := api.posts[0]["domain"].(map[string]any)
	if domain["name"] != "example" || domain["extension"] != "com" {
		t.Errorf("domain: got %v", api.posts[0]["domain"])
	}

	var got DNSZoneResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "example.com" || got.Name.ValueString() != "example" || got.Provider.ValueString() != "openprovider" {
		t.Errorf("state: got %+v", got)
	}
}

func TestDNSZoneCreateRejectsAnInvalidName(t *testing.T) {
	ctx := context.Background()
	api := newZoneAPI(t)
	r := zoneResource(api)
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := sr.Schema

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, zoneModel("example", false))}}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("create should fail")
	}
	if len(api.posts) != 0 {
		t.Errorf("the provider should send no request: %v", api.posts)
	}
}

// TestDNSZoneDeleteNeedsAllowDeletion guards the zone and all of its records:
// a destroy only forgets the zone unless allow_deletion is set.
func TestDNSZoneDeleteNeedsAllowDeletion(t *testing.T) {
	for name, tc := range map[string]struct {
		allow   bool
		deletes int
	}{"without allow_deletion": {false, 0}, "with allow_deletion": {true, 1}} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			api := newZoneAPI(t)
			r := zoneResource(api)
			var sr resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &sr)
			s := sr.Schema

			m := zoneModel("example.com", tc.allow)
			m.Provider, m.Name, m.Extension = types.StringValue("openprovider"), types.StringValue("example"), types.StringValue("com")
			m.CreationDate, m.ModificationDate, m.ID = types.StringValue(""), types.StringValue(""), types.StringValue("example.com")
			state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, m)}

			resp := &resource.DeleteResponse{}
			r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("delete: %v", resp.Diagnostics)
			}
			if len(api.deletes) != tc.deletes {
				t.Errorf("deletes: got %d, want %d", len(api.deletes), tc.deletes)
			}
		})
	}
}
