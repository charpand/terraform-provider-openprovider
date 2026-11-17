package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/dns"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type templateAPI struct {
	*httptest.Server
	posts   []map[string]any
	deletes int
	missing bool
}

func newTemplateAPI(t *testing.T) *templateAPI {
	t.Helper()
	api := &templateAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/dns/templates":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
			}
			api.posts = append(api.posts, body)
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"id":42}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/dns/templates/42" && api.missing:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"code":300,"desc":"not found"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/dns/templates/42":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"id":42,"name":"web","is_spamexperts_enabled":false,"records":[`+
				`{"id":1,"name":"www","type":"A","value":"192.0.2.9","ttl":3600},`+
				`{"id":2,"name":"","type":"MX","value":"mail.example.com","ttl":300,"prio":10}]}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/dns/templates/42":
			api.deletes++
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func templateFixture(t *testing.T, api *templateAPI) (*DNSTemplateResource, resource.SchemaResponse) {
	t.Helper()
	r := &DNSTemplateResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)
	return r, sr
}

func templateModel(t *testing.T, id types.String) DNSTemplateModel {
	t.Helper()
	records, diags := templateRecordsToSet(context.Background(), []dns.TemplateRecord{
		{Name: "www", Type: "A", Value: "192.0.2.1", TTL: 3600},
		{Name: "", Type: "MX", Value: "mail.example.com", TTL: 300, Priority: 10},
	})
	if diags.HasError() {
		t.Fatalf("records: %v", diags)
	}
	return DNSTemplateModel{ID: id, Name: types.StringValue("web"), IsSpamexpertsEnabled: types.BoolValue(false), Records: records}
}

func TestDNSTemplateCreate(t *testing.T) {
	ctx := context.Background()
	api := newTemplateAPI(t)
	r, sr := templateFixture(t, api)
	s := sr.Schema

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, templateModel(t, types.StringUnknown()))}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	if len(api.posts) != 1 {
		t.Fatalf("posts: got %d, want 1", len(api.posts))
	}
	records, _ := api.posts[0]["records"].([]any)
	if api.posts[0]["name"] != "web" || len(records) != 2 {
		t.Errorf("body: got %v", api.posts[0])
	}
	var got DNSTemplateModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "42" || len(got.Records.Elements()) != 2 {
		t.Errorf("state: got %+v", got)
	}
}

// TestDNSTemplateReadShowsDrift checks that a refresh reads what OpenProvider
// holds, so a record that changed outside Terraform shows in the plan.
func TestDNSTemplateReadShowsDrift(t *testing.T) {
	ctx := context.Background()
	api := newTemplateAPI(t)
	r, sr := templateFixture(t, api)
	s := sr.Schema

	state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, templateModel(t, types.StringValue("42")))}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got DNSTemplateModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.Records.Equal(templateModel(t, types.StringValue("42")).Records) {
		t.Error("the refresh should show the changed value of the A record")
	}
	var items []DNSTemplateRecordModel
	got.Records.ElementsAs(ctx, &items, false)
	found := false
	for _, item := range items {
		if item.Type.ValueString() == "A" && item.Value.ValueString() == "192.0.2.9" {
			found = true
		}
	}
	if !found {
		t.Errorf("records: got %+v", items)
	}
}

func TestDNSTemplateReadOfAMissingTemplateRemovesIt(t *testing.T) {
	ctx := context.Background()
	api := newTemplateAPI(t)
	api.missing = true
	r, sr := templateFixture(t, api)
	s := sr.Schema

	state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, templateModel(t, types.StringValue("42")))}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a template that is gone should leave the state")
	}
}

func TestDNSTemplateDelete(t *testing.T) {
	ctx := context.Background()
	api := newTemplateAPI(t)
	r, sr := templateFixture(t, api)
	s := sr.Schema

	state := tfsdk.State{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, templateModel(t, types.StringValue("42")))}
	resp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete: %v", resp.Diagnostics)
	}
	if api.deletes != 1 {
		t.Errorf("deletes: got %d, want 1", api.deletes)
	}
}

func TestDNSTemplateImport(t *testing.T) {
	ctx := context.Background()
	r, sr := templateFixture(t, newTemplateAPI(t))
	s := sr.Schema

	for id, wantErr := range map[string]bool{"42": false, "web": true, "": true} {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("import %q: error %v, want %v", id, resp.Diagnostics.HasError(), wantErr)
		}
	}
}
