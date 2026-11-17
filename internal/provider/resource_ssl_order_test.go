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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const issuedOrder = `{"code":0,"data":{"id":9,"product_id":7,"common_name":"www.example.com","brand_name":"Sectigo","status":"ACT",` +
	`"order_date":"2026-01-01","active_date":"2026-01-02","expiration_date":"2027-01-02","autorenew":"on",` +
	`"validation_method":"http","csr":"-----BEGIN CERTIFICATE REQUEST-----\nAAA\n-----END CERTIFICATE REQUEST-----\n",` +
	`"certificate":"CERT","intermediate_certificate":"INTERMEDIATE","root_certificate":"ROOT"}}`

type sslOrderAPI struct {
	*httptest.Server
	posts []map[string]any
}

func newSSLOrderAPI(t *testing.T, answer string) *sslOrderAPI {
	t.Helper()
	api := &sslOrderAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ssl/orders":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
			}
			api.posts = append(api.posts, body)
			_, _ = fmt.Fprint(w, answer)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ssl/orders/9":
			_, _ = fmt.Fprint(w, answer)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func sslOrderFixture(api *sslOrderAPI) (*SSLOrderResource, resource.SchemaResponse) {
	r := &SSLOrderResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test", HTTPClient: api.Client()})}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)
	return r, sr
}

func TestSSLOrderCreateSendsTheCSRAndReadsTheCertificate(t *testing.T) {
	ctx := context.Background()
	api := newSSLOrderAPI(t, issuedOrder)
	r, sr := sslOrderFixture(api)
	s := sr.Schema

	plan := SSLOrderModel{
		ID: types.Int64Unknown(), ProductID: types.Int64Value(7), CommonName: types.StringValue("www.example.com"),
		BrandName: types.StringUnknown(), Status: types.StringUnknown(), OrderDate: types.StringUnknown(), ActiveDate: types.StringUnknown(),
		ExpirationDate: types.StringUnknown(), Autorenew: types.BoolValue(true), OwnerHandle: types.StringUnknown(), AdminHandle: types.StringUnknown(),
		BillingHandle: types.StringUnknown(), TechnicalHandle: types.StringUnknown(), AdditionalDomains: types.ListNull(types.StringType),
		DomainValidationMethod: types.StringValue("http"), CSR: types.StringValue("MY CSR"),
		Certificate: types.StringUnknown(), IntermediateCertificate: types.StringUnknown(), RootCertificate: types.StringUnknown(),
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	if len(api.posts) != 1 || api.posts[0]["csr"] != "MY CSR" {
		t.Errorf("posts: got %v", api.posts)
	}
	var got SSLOrderModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.Certificate.ValueString() != "CERT" || got.IntermediateCertificate.ValueString() != "INTERMEDIATE" || got.RootCertificate.ValueString() != "ROOT" {
		t.Errorf("certificates: got %+v", got)
	}
	if got.CSR.ValueString() != "MY CSR" {
		t.Errorf("the state should keep the configured CSR, got %q", got.CSR.ValueString())
	}
}

func TestSSLOrderCreateWithoutACSRTakesTheGeneratedOne(t *testing.T) {
	ctx := context.Background()
	api := newSSLOrderAPI(t, issuedOrder)
	r, sr := sslOrderFixture(api)
	s := sr.Schema

	plan := SSLOrderModel{
		ID: types.Int64Unknown(), ProductID: types.Int64Value(7), CommonName: types.StringValue("www.example.com"),
		BrandName: types.StringUnknown(), Status: types.StringUnknown(), OrderDate: types.StringUnknown(), ActiveDate: types.StringUnknown(),
		ExpirationDate: types.StringUnknown(), Autorenew: types.BoolValue(false), OwnerHandle: types.StringUnknown(), AdminHandle: types.StringUnknown(),
		BillingHandle: types.StringUnknown(), TechnicalHandle: types.StringUnknown(), AdditionalDomains: types.ListNull(types.StringType),
		DomainValidationMethod: types.StringValue("dns"), CSR: types.StringUnknown(),
		Certificate: types.StringUnknown(), IntermediateCertificate: types.StringUnknown(), RootCertificate: types.StringUnknown(),
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if _, ok := api.posts[0]["csr"]; ok {
		t.Errorf("an unset CSR should not be sent: %v", api.posts[0])
	}
	var got SSLOrderModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.CSR.IsNull() || got.CSR.IsUnknown() {
		t.Errorf("the state should hold the generated CSR, got %v", got.CSR)
	}
}

// TestSSLOrderImportReadsTheArguments checks that an order that is imported by
// its id gets the arguments that a configuration sets from the order itself.
func TestSSLOrderImportReadsTheArguments(t *testing.T) {
	ctx := context.Background()
	api := newSSLOrderAPI(t, issuedOrder)
	r, sr := sslOrderFixture(api)
	s := sr.Schema

	imp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "9"}, imp)
	if imp.Diagnostics.HasError() {
		t.Fatalf("import: %v", imp.Diagnostics)
	}

	resp := &resource.ReadResponse{State: imp.State}
	r.Read(ctx, resource.ReadRequest{State: imp.State}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got SSLOrderModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueInt64() != 9 || got.ProductID.ValueInt64() != 7 || got.CommonName.ValueString() != "www.example.com" {
		t.Errorf("arguments: got %+v", got)
	}
	if got.DomainValidationMethod.ValueString() != "http" || got.Certificate.ValueString() != "CERT" || got.Status.ValueString() != "ACT" {
		t.Errorf("order: got %+v", got)
	}
	if len(api.posts) != 0 {
		t.Errorf("an import must not order: %v", api.posts)
	}
}

func TestSSLOrderImportRejectsAnInvalidIdentifier(t *testing.T) {
	ctx := context.Background()
	r, sr := sslOrderFixture(newSSLOrderAPI(t, issuedOrder))
	s := sr.Schema
	for _, id := range []string{"", "abc", "0", "-3", "1.5"} {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("identifier %q: the import should fail", id)
		}
	}
}

// TestSSLOrderCSRChangeReplaces guards the account: a different CSR orders a
// new certificate, so only a real change may do that, not whitespace and not
// the CSR that an import read into the state.
func TestSSLOrderCSRChangeReplaces(t *testing.T) {
	for name, tc := range map[string]struct {
		state, plan types.String
		want        bool
	}{
		"a different CSR":     {types.StringValue("AAA"), types.StringValue("BBB"), true},
		"the same CSR":        {types.StringValue("AAA"), types.StringValue("AAA"), false},
		"whitespace only":     {types.StringValue("AAA\n"), types.StringValue("AAA"), false},
		"no CSR in the state": {types.StringNull(), types.StringValue("AAA"), false},
		"an unknown plan":     {types.StringValue("AAA"), types.StringUnknown(), false},
	} {
		t.Run(name, func(t *testing.T) {
			resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
			csrChanged(context.Background(), planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
			if resp.RequiresReplace != tc.want {
				t.Errorf("replace: got %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}
