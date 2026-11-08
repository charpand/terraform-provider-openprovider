package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// restartAPI stands in for an account that holds example.com with the id 1
// and the status in `status`. An empty status means that the account has no
// such domain. `answer` is the body of the answer to a restart, and
// `failure` is its HTTP status when it is set.
type restartAPI struct {
	*httptest.Server
	status  string
	answer  string
	failure int
	posts   []map[string]any
}

func newRestartAPI(t *testing.T, status string) *restartAPI {
	t.Helper()
	api := &restartAPI{status: status, answer: `{"code":0,"data":{"success":true}}`}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains" && api.status == "":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[],"total":0}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[{"id":1,"status":"`+api.status+`",`+
				`"domain":{"name":"example","extension":"com"}}],"total":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/domains/1/last-operation/restart":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode restart: %v", err)
			}
			api.posts = append(api.posts, body)
			if api.failure != 0 {
				w.WriteHeader(api.failure)
			}
			_, _ = fmt.Fprint(w, api.answer)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func createRestart(t *testing.T, api *restartAPI, authCode types.String) *resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	r := &DomainTransferRestartResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	plan := DomainTransferRestartModel{
		ID:       types.StringUnknown(),
		Domain:   types.StringValue("example.com"),
		AuthCode: authCode,
		Triggers: types.MapNull(types.StringType),
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
	}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	return resp
}

func TestDomainTransferRestartSendsTheOperation(t *testing.T) {
	api := newRestartAPI(t, "FAI")
	resp := createRestart(t, api, types.StringValue("secret"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	want := []map[string]any{{
		"id":        float64(1),
		"domain":    map[string]any{"name": "example", "extension": "com"},
		"auth_code": "secret",
	}}
	if !reflect.DeepEqual(api.posts, want) {
		t.Errorf("posts: got %v, want %v", api.posts, want)
	}
	var got DomainTransferRestartModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "example.com" {
		t.Errorf("id: got %v, want example.com", got.ID)
	}
}

func TestDomainTransferRestartWithoutAuthCode(t *testing.T) {
	api := newRestartAPI(t, "FAI")
	if resp := createRestart(t, api, types.StringNull()); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if len(api.posts) != 1 {
		t.Fatalf("posts: got %d, want 1", len(api.posts))
	}
	if _, ok := api.posts[0]["auth_code"]; ok {
		t.Errorf("the request should have no auth code: %v", api.posts[0])
	}
}

// TestDomainTransferRestartOnlyAfterAFailure guards against a restart of a
// different operation, because the API restarts the last operation of the
// domain, whatever it is.
func TestDomainTransferRestartOnlyAfterAFailure(t *testing.T) {
	for _, status := range []string{"", "ACT", "REQ", "PEN", "DEL"} {
		t.Run("status "+status, func(t *testing.T) {
			api := newRestartAPI(t, status)
			resp := createRestart(t, api, types.StringNull())
			if !resp.Diagnostics.HasError() {
				t.Fatal("the create should fail")
			}
			if len(api.posts) != 0 {
				t.Errorf("the provider should send no restart: %v", api.posts)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("the create should keep no state")
			}
		})
	}
}

func TestDomainTransferRestartRefused(t *testing.T) {
	for name, tc := range map[string]struct {
		answer  string
		failure int
	}{
		"code in a 2xx": {answer: `{"code":399,"desc":"refused"}`},
		"no success":    {answer: `{"code":0,"data":{"success":false}}`},
		"server error":  {answer: `{}`, failure: http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			api := newRestartAPI(t, "FAI")
			api.answer, api.failure = tc.answer, tc.failure
			resp := createRestart(t, api, types.StringNull())
			if !resp.Diagnostics.HasError() {
				t.Fatal("the create should fail")
			}
			if !resp.State.Raw.IsNull() {
				t.Error("the create should keep no state")
			}
		})
	}
}

// TestDomainTransferRestartReplaces checks that a change of each configured
// attribute replaces the resource, so that the change restarts the transfer.
func TestDomainTransferRestartReplaces(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewDomainTransferRestartResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	replace := stringplanmodifier.RequiresReplace().Description(ctx)
	replaceMap := mapplanmodifier.RequiresReplace().Description(ctx)
	for name, attribute := range resp.Schema.Attributes {
		if name == "id" {
			continue
		}
		var modifiers []string
		switch a := attribute.(type) {
		case interface{ StringPlanModifiers() []planmodifier.String }:
			for _, m := range a.StringPlanModifiers() {
				modifiers = append(modifiers, m.Description(ctx))
			}
		case interface{ MapPlanModifiers() []planmodifier.Map }:
			for _, m := range a.MapPlanModifiers() {
				modifiers = append(modifiers, m.Description(ctx))
			}
		}
		joined := strings.Join(modifiers, ",")
		if !strings.Contains(joined, replace) && !strings.Contains(joined, replaceMap) {
			t.Errorf("attribute %s should replace the resource, modifiers: %v", name, modifiers)
		}
	}
}
