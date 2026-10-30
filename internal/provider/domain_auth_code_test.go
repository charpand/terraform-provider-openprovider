package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// authCodeAPI stands in for an account that holds example.com with the id 1.
// It keeps each auth code request it receives. When `unlisted` is set, the
// account holds no domain. `code` is the auth code that it gives.
type authCodeAPI struct {
	*httptest.Server
	unlisted bool
	code     string
	requests []string
}

func newAuthCodeAPI(t *testing.T) *authCodeAPI {
	t.Helper()
	api := &authCodeAPI{code: "secret"}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains" && api.unlisted:
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[],"total":0}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[{"id":1,"status":"ACT",`+
				`"domain":{"name":"example","extension":"com"}}],"total":1}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains/1/authcode":
			api.requests = append(api.requests, "get "+r.URL.RawQuery)
			_, _ = fmt.Fprintf(w, `{"code":0,"data":{"auth_code":%q}}`, api.code)
		case r.Method == http.MethodPost && r.URL.Path == "/v1beta/domains/1/authcode/reset":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read reset: %v", err)
			}
			var reset map[string]any
			if err := json.Unmarshal(body, &reset); err != nil {
				t.Errorf("decode reset %s: %v", body, err)
			}
			api.requests = append(api.requests, fmt.Sprint("reset ", reset["auth_code_type"]))
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"auth_code":"new"}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return api
}

func (api *authCodeAPI) client() *client.Client {
	return client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})
}

// rawOf gives the Terraform value of `m` under the schema of `state`.
func rawOf(ctx context.Context, t *testing.T, state tfsdk.State, m any) tftypes.Value {
	t.Helper()
	state.Raw = tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)
	if diags := state.Set(ctx, m); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return state.Raw
}

func openAuthCode(ctx context.Context, t *testing.T, api *authCodeAPI, config DomainAuthCodeModel) (DomainAuthCodeModel, *ephemeral.OpenResponse) {
	t.Helper()
	r := &DomainAuthCodeEphemeralResource{client: api.client()}
	var schemaResp ephemeral.SchemaResponse
	r.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	resp := &ephemeral.OpenResponse{
		Result: tfsdk.EphemeralResultData{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
	}
	r.Open(ctx, ephemeral.OpenRequest{Config: tfsdk.Config{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, config)}}, resp)
	var got DomainAuthCodeModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.Result.Get(ctx, &got); diags.HasError() {
			t.Fatalf("get: %v", diags)
		}
	}
	return got, resp
}

// TestDomainAuthCodeOpen covers a read of the auth code. The read takes the
// id of the domain from the account, sends the configured type, and gives
// the type that it read.
func TestDomainAuthCodeOpen(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		authCodeType types.String
		wantRequest  string
		wantType     string
	}{
		"not set":  {authCodeType: types.StringNull(), wantRequest: "get auth_code_type=external", wantType: "external"},
		"internal": {authCodeType: types.StringValue("internal"), wantRequest: "get auth_code_type=internal", wantType: "internal"},
	} {
		t.Run(name, func(t *testing.T) {
			api := newAuthCodeAPI(t)
			defer api.Close()

			got, resp := openAuthCode(ctx, t, api, DomainAuthCodeModel{
				Domain:       types.StringValue("example.com"),
				AuthCodeType: tc.authCodeType,
				AuthCode:     types.StringNull(),
			})
			if resp.Diagnostics.HasError() {
				t.Fatalf("open: %v", resp.Diagnostics)
			}
			if fmt.Sprint(api.requests) != fmt.Sprint([]string{tc.wantRequest}) {
				t.Errorf("requests: got %v, want %v", api.requests, []string{tc.wantRequest})
			}
			if got.AuthCode.ValueString() != "secret" {
				t.Errorf("auth_code: got %v, want secret", got.AuthCode)
			}
			if got.AuthCodeType.ValueString() != tc.wantType {
				t.Errorf("auth_code_type: got %v, want %s", got.AuthCodeType, tc.wantType)
			}
		})
	}
}

// TestDomainAuthCodeOpenFails covers a domain that the account does not hold
// and an empty auth code. Each fails the read.
func TestDomainAuthCodeOpenFails(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		unlisted  bool
		code      string
		wantError string
	}{
		"not held":   {unlisted: true, code: "secret", wantError: "Domain Not Found"},
		"empty code": {code: "", wantError: "Error Reading Auth Code"},
	} {
		t.Run(name, func(t *testing.T) {
			api := newAuthCodeAPI(t)
			api.unlisted = tc.unlisted
			api.code = tc.code
			defer api.Close()

			_, resp := openAuthCode(ctx, t, api, DomainAuthCodeModel{
				Domain:       types.StringValue("example.com"),
				AuthCodeType: types.StringNull(),
				AuthCode:     types.StringNull(),
			})
			if !resp.Diagnostics.HasError() {
				t.Fatal("open: expected an error")
			}
			if summary := resp.Diagnostics.Errors()[0].Summary(); summary != tc.wantError {
				t.Errorf("error: got %q, want %q", summary, tc.wantError)
			}
		})
	}
}

// TestDomainAuthCodeAttributes checks that the auth code is sensitive.
func TestDomainAuthCodeAttributes(t *testing.T) {
	var resp ephemeral.SchemaResponse
	NewDomainAuthCodeEphemeralResource().Schema(context.Background(), ephemeral.SchemaRequest{}, &resp)
	if !resp.Schema.Attributes["auth_code"].IsSensitive() {
		t.Error("auth_code should be sensitive")
	}
}

// TestAuthCodeTypeValidator checks that only the types of OpenProvider pass.
func TestAuthCodeTypeValidator(t *testing.T) {
	ctx := context.Background()
	for value, wantError := range map[string]bool{"external": false, "internal": false, "other": true} {
		resp := &validator.StringResponse{}
		authCodeTypeValidator{}.ValidateString(ctx, validator.StringRequest{
			Path:        path.Root("auth_code_type"),
			ConfigValue: types.StringValue(value),
		}, resp)
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("%s: error %v, want %v", value, resp.Diagnostics.HasError(), wantError)
		}
	}
}

// TestDomainAuthCodeResetCreate covers a create, which resets the auth code
// with the configured type and does not keep the new code.
func TestDomainAuthCodeResetCreate(t *testing.T) {
	ctx := context.Background()
	api := newAuthCodeAPI(t)
	defer api.Close()
	r := &DomainAuthCodeResetResource{client: api.client()}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	plan := DomainAuthCodeResetModel{
		ID:           types.StringUnknown(),
		Domain:       types.StringValue("example.com"),
		AuthCodeType: types.StringValue("internal"),
		Triggers:     types.MapValueMust(types.StringType, map[string]attr.Value{"export": types.StringValue("1")}),
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
	}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: rawOf(ctx, t, tfsdk.State{Schema: s}, plan)}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if fmt.Sprint(api.requests) != fmt.Sprint([]string{"reset internal"}) {
		t.Errorf("requests: got %v, want [reset internal]", api.requests)
	}
	var got DomainAuthCodeResetModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	if got.ID.ValueString() != "example.com" {
		t.Errorf("id: got %v, want example.com", got.ID)
	}
	if strings.Contains(resp.State.Raw.String(), "new") {
		t.Error("the state should not hold the new auth code")
	}
}

// TestDomainAuthCodeResetReplaces checks that a change of each configured
// attribute replaces the resource, so that the change resets the auth code.
func TestDomainAuthCodeResetReplaces(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewDomainAuthCodeResetResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	replace := stringplanmodifier.RequiresReplace().Description(ctx)
	for name, attribute := range resp.Schema.Attributes {
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
		replaces := strings.Contains(strings.Join(modifiers, "\n"), replace)
		if replaces != (name != "id") {
			t.Errorf("%s: replaces %v, want %v", name, replaces, name != "id")
		}
	}
	if mapplanmodifier.RequiresReplace().Description(ctx) != replace {
		t.Fatal("the map and string replace modifiers have different descriptions")
	}
}
