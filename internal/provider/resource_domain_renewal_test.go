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

// renewalAPI stands in for an account that holds example.com with the id 1,
// the status in `status` and the renewal allowed or not. An empty status
// means that the account has no such domain.
type renewalAPI struct {
	*httptest.Server
	status   string
	canRenew bool
	posts    []map[string]any
}

func newRenewalAPI(t *testing.T, status string, canRenew bool) *renewalAPI {
	t.Helper()
	api := &renewalAPI{status: status, canRenew: canRenew}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains" && api.status == "":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[],"total":0}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains":
			_, _ = fmt.Fprintf(w, `{"code":0,"data":{"results":[{"id":1,"status":%q,"can_renew":%t,`+
				`"expiration_date":"2027-01-01 00:00:00","domain":{"name":"example","extension":"com"}}],"total":1}}`, api.status, api.canRenew)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/domains/1/renew":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode renew: %v", err)
			}
			api.posts = append(api.posts, body)
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"status":"ACT"}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func createRenewal(t *testing.T, api *renewalAPI, period int64) *resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	r := &DomainRenewalResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	plan := DomainRenewalModel{
		ID:             types.StringUnknown(),
		Domain:         types.StringValue("example.com"),
		Period:         types.Int64Value(period),
		Triggers:       types.MapNull(types.StringType),
		Status:         types.StringUnknown(),
		ExpirationDate: types.StringUnknown(),
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	return resp
}

func TestDomainRenewalSendsTheRenewal(t *testing.T) {
	api := newRenewalAPI(t, "ACT", true)
	resp := createRenewal(t, api, 2)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	want := []map[string]any{{
		"id":     float64(1),
		"domain": map[string]any{"name": "example", "extension": "com"},
		"period": float64(2),
	}}
	if !reflect.DeepEqual(api.posts, want) {
		t.Errorf("posts: got %v, want %v", api.posts, want)
	}
	var got DomainRenewalModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "example.com" || got.Status.ValueString() != "ACT" || got.ExpirationDate.ValueString() != "2027-01-01 00:00:00" {
		t.Errorf("state: got %+v", got)
	}
}

// TestDomainRenewalOnlyWhenAllowed guards the account: a renewal is charged,
// so the provider sends none when the domain is missing, not active, or when
// the API does not allow a renewal now.
func TestDomainRenewalOnlyWhenAllowed(t *testing.T) {
	for name, tc := range map[string]struct {
		status   string
		canRenew bool
	}{
		"no such domain": {"", true},
		"expired":        {"EXP", true},
		"deleted":        {"DEL", true},
		"not renewable":  {"ACT", false},
	} {
		t.Run(name, func(t *testing.T) {
			api := newRenewalAPI(t, tc.status, tc.canRenew)
			resp := createRenewal(t, api, 1)
			if !resp.Diagnostics.HasError() {
				t.Fatal("the create should fail")
			}
			if len(api.posts) != 0 {
				t.Errorf("the provider should send no renewal: %v", api.posts)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("the create should keep no state")
			}
		})
	}
}

func TestDomainRenewalRejectsAPeriodOutOfRange(t *testing.T) {
	for _, period := range []int64{0, 11} {
		api := newRenewalAPI(t, "ACT", true)
		if resp := createRenewal(t, api, period); !resp.Diagnostics.HasError() {
			t.Errorf("period %d: the create should fail", period)
		}
		if len(api.posts) != 0 {
			t.Errorf("period %d: the provider should send no renewal", period)
		}
	}
}
