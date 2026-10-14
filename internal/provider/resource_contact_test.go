package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const contactJSON = `{"code":0,"data":{"id":99,"username":"ci","email":"ci@example.com","role":"admin","locale":"en_GB",
"is_active":true,"api_access_enabled":true,
"name":{"first_name":"C","last_name":"I","full_name":"C I"},
"phone":{"country_code":"+31","area_code":"20","subscriber_number":"1234567"},
"address":{"street":"Main","number":"1","zipcode":"1000AA","city":"Amsterdam","country":"NL"},
"secret_key":"never-in-state"}}`

type fakeContactAPI struct {
	mu      sync.Mutex
	bodies  map[string][]string // method -> request bodies
	getBody string
	getCode int
}

func (f *fakeContactAPI) handler(t *testing.T, wantPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		if f.bodies == nil {
			f.bodies = map[string][]string{}
		}
		f.bodies[r.Method] = append(f.bodies[r.Method], string(body))
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/v1beta/contacts" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"id":99,"token":"tok"}}`))
		case http.MethodGet:
			if r.URL.Path != wantPath {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			if f.getCode != 0 {
				w.WriteHeader(f.getCode)
				_, _ = w.Write([]byte(`{"code":4}`))
				return
			}
			b := f.getBody
			if b == "" {
				b = contactJSON
			}
			_, _ = w.Write([]byte(b))
		default:
			_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
		}
	}
}

func contactResource(server *httptest.Server) *ContactResource {
	return &ContactResource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "t", HTTPClient: server.Client()})}
}

func contactState(ctx context.Context, t *testing.T, r *ContactResource, m ContactResourceModel) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, m); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return state
}

// unplanned is a model the way a plan holds a contact whose optional values
// are all left out: unknown wherever the API decides.
func unplanned() ContactResourceModel {
	return ContactResourceModel{
		ID: types.StringUnknown(), Username: types.StringValue("ci"), Password: types.StringNull(),
		Email: types.StringValue("ci@example.com"), Role: types.StringUnknown(), CompanyName: types.StringUnknown(),
		Locale: types.StringUnknown(), Comments: types.StringUnknown(), VAT: types.StringUnknown(),
		IsActive: types.BoolUnknown(), APIAccessEnabled: types.BoolUnknown(),
		Name:      types.ObjectUnknown(contactNameTypes),
		Phone:     types.ObjectUnknown(contactPhoneTypes),
		Address:   types.ObjectUnknown(contactAddressTypes),
		OnDestroy: types.StringValue("retain"),
	}
}

func TestContactCreateSendsPasswordAndKeepsItOutOfState(t *testing.T) {
	ctx := context.Background()
	f := &fakeContactAPI{}
	server := httptest.NewServer(f.handler(t, "/v1beta/contacts/99"))
	defer server.Close()

	res := contactResource(server)
	plan := contactState(ctx, t, res, unplanned())
	withPassword := unplanned()
	withPassword.Password = types.StringValue("s3cret")
	config := contactState(ctx, t, res, withPassword)

	resp := &resource.CreateResponse{State: plan}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(plan), Config: tfsdk.Config(config)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	var sent map[string]any
	if err := json.Unmarshal([]byte(f.bodies["POST"][0]), &sent); err != nil {
		t.Fatal(err)
	}
	if sent["username"] != "ci" || sent["password"] != "s3cret" || sent["email"] != "ci@example.com" {
		t.Errorf("unexpected create body %v", sent)
	}
	for _, k := range []string{"role", "name", "phone", "address", "is_active"} {
		if _, ok := sent[k]; ok {
			t.Errorf("%s was sent although the plan left it unknown", k)
		}
	}

	var got ContactResourceModel
	resp.State.Get(ctx, &got)
	if got.ID.ValueString() != "99" || got.Role.ValueString() != "admin" || !got.IsActive.ValueBool() {
		t.Errorf("unexpected state %+v", got)
	}
	if !got.Password.IsNull() {
		t.Error("password must not be in state")
	}
	if strings.Contains(got.Name.String(), "full_name") || got.Name.Attributes()["first_name"].String() != `"C"` {
		t.Errorf("unexpected name %v", got.Name)
	}
	if got.Phone.Attributes()["number"].String() != `"1234567"` || got.Address.Attributes()["city"].String() != `"Amsterdam"` {
		t.Errorf("unexpected phone or address: %v %v", got.Phone, got.Address)
	}
	if got.Address.Attributes()["state"].IsNull() != true {
		t.Error("a field the API leaves out should be null")
	}
}

func TestContactCreateRequiresPassword(t *testing.T) {
	ctx := context.Background()
	f := &fakeContactAPI{}
	server := httptest.NewServer(f.handler(t, ""))
	defer server.Close()

	res := contactResource(server)
	plan := contactState(ctx, t, res, unplanned())
	resp := &resource.CreateResponse{State: plan}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(plan), Config: tfsdk.Config(plan)}, resp)
	if !resp.Diagnostics.HasError() || len(f.bodies["POST"]) != 0 {
		t.Fatalf("expected an error and no API call, got %v / %v", resp.Diagnostics, f.bodies)
	}
}

func TestContactReadMapsAndDropsSecrets(t *testing.T) {
	ctx := context.Background()
	f := &fakeContactAPI{}
	server := httptest.NewServer(f.handler(t, "/v1beta/contacts/99"))
	defer server.Close()

	res := contactResource(server)
	// the shape right after an import: only the id and on_destroy are set
	prior := ContactResourceModel{
		ID: types.StringValue("99"), Username: types.StringNull(), Password: types.StringNull(),
		Email: types.StringNull(), Role: types.StringNull(), CompanyName: types.StringNull(), Locale: types.StringNull(),
		Comments: types.StringNull(), VAT: types.StringNull(), IsActive: types.BoolNull(), APIAccessEnabled: types.BoolNull(),
		Name: types.ObjectNull(contactNameTypes), Phone: types.ObjectNull(contactPhoneTypes), Address: types.ObjectNull(contactAddressTypes),
		OnDestroy: types.StringValue("retain"),
	}
	state := contactState(ctx, t, res, prior)
	resp := &resource.ReadResponse{State: state}
	res.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got ContactResourceModel
	resp.State.Get(ctx, &got)
	if got.Username.ValueString() != "ci" || got.Email.ValueString() != "ci@example.com" || got.Locale.ValueString() != "en_GB" {
		t.Errorf("unexpected state %+v", got)
	}
	if strings.Contains(resp.State.Raw.String(), "never-in-state") {
		t.Error("the secret key leaked into state")
	}
}

func TestContactReadGoneRemovesResource(t *testing.T) {
	ctx := context.Background()
	f := &fakeContactAPI{getCode: http.StatusNotFound}
	server := httptest.NewServer(f.handler(t, "/v1beta/contacts/99"))
	defer server.Close()

	res := contactResource(server)
	m := unplanned()
	m.ID = types.StringValue("99")
	state := contactState(ctx, t, res, m)
	resp := &resource.ReadResponse{State: state}
	res.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Fatalf("expected the resource to be removed, got %v", resp.Diagnostics)
	}
}

func TestContactUpdateSendsPlannedFieldsOnly(t *testing.T) {
	ctx := context.Background()
	f := &fakeContactAPI{}
	server := httptest.NewServer(f.handler(t, "/v1beta/contacts/99"))
	defer server.Close()

	res := contactResource(server)
	m := unplanned()
	m.ID = types.StringValue("99")
	m.Email = types.StringValue("new@example.com")
	m.Role = types.StringValue("admin")
	m.Locale = types.StringValue("en_GB")
	m.IsActive = types.BoolValue(true)
	m.Name = types.ObjectValueMust(contactNameTypes, map[string]attr.Value{
		"first_name": types.StringValue("C"), "last_name": types.StringValue("I"),
		"initials": types.StringUnknown(), "prefix": types.StringNull(),
	})
	plan := contactState(ctx, t, res, m)
	resp := &resource.UpdateResponse{State: plan}
	res.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan(plan), State: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	want := `{"email":"new@example.com","role":"admin","locale":"en_GB","is_active":true,"name":{"first_name":"C","last_name":"I"}}`
	if got := f.bodies["PUT"][0]; got != want {
		t.Errorf("PUT body = %s, want %s", got, want)
	}
}

func TestContactDelete(t *testing.T) {
	for _, tc := range []struct {
		onDestroy string
		wantCalls int
	}{{"retain", 0}, {"delete", 1}} {
		t.Run(tc.onDestroy, func(t *testing.T) {
			ctx := context.Background()
			f := &fakeContactAPI{}
			server := httptest.NewServer(f.handler(t, ""))
			defer server.Close()

			res := contactResource(server)
			m := unplanned()
			m.ID = types.StringValue("99")
			m.OnDestroy = types.StringValue(tc.onDestroy)
			state := contactState(ctx, t, res, m)
			resp := &resource.DeleteResponse{State: state}
			res.Delete(ctx, resource.DeleteRequest{State: state}, resp)
			if resp.Diagnostics.HasError() || len(f.bodies["DELETE"]) != tc.wantCalls {
				t.Fatalf("delete calls = %d, want %d (%v)", len(f.bodies["DELETE"]), tc.wantCalls, resp.Diagnostics)
			}
			if tc.onDestroy == "retain" && resp.Diagnostics.WarningsCount() != 1 {
				t.Error("expected a warning that the contact stays")
			}
		})
	}
}

func TestContactImport(t *testing.T) {
	ctx := context.Background()
	res := &ContactResource{}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	newState := func() tfsdk.State {
		return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	}

	ok := &resource.ImportStateResponse{State: newState()}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "99"}, ok)
	if ok.Diagnostics.HasError() {
		t.Fatalf("import: %v", ok.Diagnostics)
	}
	for _, bad := range []string{"abc", "0", ""} {
		resp := &resource.ImportStateResponse{State: newState()}
		res.ImportState(ctx, resource.ImportStateRequest{ID: bad}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("expected an error for import id %q", bad)
		}
	}
}

func TestContactSchemaIsValid(t *testing.T) {
	ctx := context.Background()
	for name, r := range map[string]resource.Resource{"contact": &ContactResource{}, "whitelist": &RCPIPWhitelistResource{}} {
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: %v", name, diags)
		}
	}
}
