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
)

// settingsAPI stands in for an account that holds one active domain with the
// WHOIS privacy, comments and nameservers of the fields below. A create or an
// update takes the settings from its body, and the listing shows what the
// account has. It keeps the body of each order and update.
type settingsAPI struct {
	*httptest.Server
	whois       bool
	allowed     bool
	comments    string
	nameservers string
	posts       []map[string]any
	puts        []map[string]any
}

func newSettingsAPI(t *testing.T) *settingsAPI {
	t.Helper()
	api := &settingsAPI{allowed: true, nameservers: `[]`}
	domain := func() string {
		return fmt.Sprintf(`{"id":1,"status":"ACT","autorenew":"off","owner_handle":"XX000001-NL","is_locked":false,`+
			`"is_private_whois_enabled":%t,"is_private_whois_allowed":%t,"comments":%q,"name_servers":%s,`+
			`"domain":{"name":"example","extension":"com"}}`, api.whois, api.allowed, api.comments, api.nameservers)
	}
	take := func(body map[string]any) {
		if v, ok := body["is_private_whois_enabled"].(bool); ok {
			api.whois = v
		}
		if v, ok := body["comments"].(string); ok {
			api.comments = v
		}
		if v, ok := body["name_servers"]; ok {
			raw, _ := json.Marshal(v)
			api.nameservers = string(raw)
		}
	}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/domains":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode order: %v", err)
			}
			api.posts = append(api.posts, body)
			take(body)
			_, _ = fmt.Fprint(w, `{"code":0,"data":`+domain()+`}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/domains":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+domain()+`],"total":1}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/v1/domains/1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode update: %v", err)
			}
			api.puts = append(api.puts, body)
			take(body)
			_, _ = fmt.Fprint(w, `{"code":0,"data":`+domain()+`}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func (a *settingsAPI) resource() *DomainResource {
	return &DomainResource{client: client.NewClient(client.Config{BaseURL: a.URL, Token: "test"})}
}

func nameserverList(t *testing.T, servers ...NameserverModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: nameserverAttrTypes}, servers)
	if diags.HasError() {
		t.Fatalf("list: %v", diags)
	}
	return list
}

func settingsBase() DomainModel {
	return DomainModel{
		ID:                    types.StringValue("example.com"),
		Domain:                types.StringValue("example.com"),
		Status:                types.StringValue("ACT"),
		Autorenew:             types.BoolValue(false),
		OwnerHandle:           types.StringValue("XX000001-NL"),
		DnssecKeys:            types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}),
		Nameservers:           types.ListNull(types.ObjectType{AttrTypes: nameserverAttrTypes}),
		AdditionalData:        types.MapNull(types.StringType),
		OnDestroy:             types.StringValue(onDestroyRetain),
		IsPrivateWhoisEnabled: types.BoolValue(false),
		IsPrivateWhoisAllowed: types.BoolValue(true),
		Comments:              types.StringValue(""),
	}
}

func updateWith(ctx context.Context, t *testing.T, r *DomainResource, prior, plan DomainModel) (DomainModel, resource.UpdateResponse) {
	t.Helper()
	priorState := stateOf(ctx, t, r, prior)
	resp := resource.UpdateResponse{State: priorState}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
		State: priorState,
	}, &resp)
	var got DomainModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("get: %v", diags)
		}
	}
	return got, resp
}

func TestDomainResourceCreateSendsTheSettings(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	r := api.resource()

	plan := settingsBase()
	plan.ID, plan.Status = types.StringUnknown(), types.StringUnknown()
	plan.Period, plan.MaxCost = types.Int64Value(1), types.Int64Null()
	plan.IsPrivateWhoisEnabled = types.BoolValue(true)
	plan.IsPrivateWhoisAllowed = types.BoolUnknown()
	plan.Comments = types.StringValue("renewed by terraform")
	plan.Nameservers = nameserverList(t,
		NameserverModel{Name: types.StringValue("ns1.example.net"), IP: types.StringNull(), IP6: types.StringNull()},
		NameserverModel{Name: types.StringValue("ns2.example.net"), IP: types.StringValue("192.0.2.2"), IP6: types.StringNull()},
	)

	got := createDomain(ctx, t, r, plan)

	if len(api.posts) != 1 {
		t.Fatalf("orders: got %d, want 1", len(api.posts))
	}
	post := api.posts[0]
	if post["is_private_whois_enabled"] != true || post["comments"] != "renewed by terraform" {
		t.Errorf("order: got %v", post)
	}
	servers, _ := post["name_servers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("name_servers: got %v", post["name_servers"])
	}
	if second, _ := servers[1].(map[string]any); second["name"] != "ns2.example.net" || second["ip"] != "192.0.2.2" {
		t.Errorf("second nameserver: got %v", servers[1])
	}
	if !got.IsPrivateWhoisEnabled.ValueBool() || got.Comments.ValueString() != "renewed by terraform" || !got.IsPrivateWhoisAllowed.ValueBool() {
		t.Errorf("state: got %+v", got)
	}
}

// TestDomainResourceCreateWithoutSettingsSendsNone guards the existing
// configurations: a domain that sets none of the new arguments orders as it
// did, and the state takes what OpenProvider holds.
func TestDomainResourceCreateWithoutSettingsSendsNone(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	api.comments = "from the registry"
	r := api.resource()

	plan := settingsBase()
	plan.ID, plan.Status = types.StringUnknown(), types.StringUnknown()
	plan.Period, plan.MaxCost = types.Int64Value(1), types.Int64Null()
	plan.IsPrivateWhoisEnabled, plan.IsPrivateWhoisAllowed, plan.Comments = types.BoolUnknown(), types.BoolUnknown(), types.StringUnknown()

	got := createDomain(ctx, t, r, plan)

	for _, key := range []string{"is_private_whois_enabled", "comments", "name_servers"} {
		if _, ok := api.posts[0][key]; ok {
			t.Errorf("the order should not send %s: %v", key, api.posts[0])
		}
	}
	if got.Comments.ValueString() != "from the registry" || got.IsPrivateWhoisEnabled.ValueBool() || !got.IsPrivateWhoisAllowed.ValueBool() {
		t.Errorf("state: got %+v", got)
	}
	if !got.Nameservers.IsNull() {
		t.Errorf("nameservers that the configuration does not set stay null, got %v", got.Nameservers)
	}
}

func TestDomainResourceUpdateSendsOnlyTheChangedSettings(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		change   func(*DomainModel)
		wantKeys map[string]any
	}{
		"WHOIS privacy on": {
			change:   func(m *DomainModel) { m.IsPrivateWhoisEnabled = types.BoolValue(true) },
			wantKeys: map[string]any{"is_private_whois_enabled": true},
		},
		"comments": {
			change:   func(m *DomainModel) { m.Comments = types.StringValue("hello") },
			wantKeys: map[string]any{"comments": "hello"},
		},
		"nameservers": {
			change: func(m *DomainModel) {
				m.Nameservers = nameserverList(t, NameserverModel{Name: types.StringValue("ns1.example.net"), IP: types.StringNull(), IP6: types.StringNull()})
			},
			wantKeys: map[string]any{"name_servers": []any{map[string]any{"name": "ns1.example.net"}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := newSettingsAPI(t)
			r := api.resource()
			plan := settingsBase()
			tc.change(&plan)

			if _, resp := updateWith(ctx, t, r, settingsBase(), plan); resp.Diagnostics.HasError() {
				t.Fatalf("update: %v", resp.Diagnostics)
			}
			if len(api.puts) != 1 {
				t.Fatalf("updates: got %d, want 1", len(api.puts))
			}
			// Only the changed setting is sent.
			if !reflect.DeepEqual(api.puts[0], tc.wantKeys) {
				t.Errorf("update body: got %v, want %v", api.puts[0], tc.wantKeys)
			}
		})
	}
}

func TestDomainResourceUpdateRemovesTheComments(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	api.comments = "old"
	r := api.resource()

	prior := settingsBase()
	prior.Comments = types.StringValue("old")
	plan := settingsBase()
	plan.Comments = types.StringValue("")

	got, resp := updateWith(ctx, t, r, prior, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if comments, ok := api.puts[0]["comments"]; !ok || comments != "" {
		t.Errorf("an empty comment should be sent to remove the comments: %v", api.puts[0])
	}
	if got.Comments.ValueString() != "" {
		t.Errorf("comments: got %q", got.Comments.ValueString())
	}
}

func TestDomainResourceUpdateWithUnsetSettingsSendsNothing(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	api.whois, api.comments = true, "kept"
	r := api.resource()

	prior := settingsBase()
	prior.IsPrivateWhoisEnabled, prior.Comments = types.BoolValue(true), types.StringValue("kept")
	// An unset argument plans the value of the state (UseStateForUnknown).
	plan := prior

	if _, resp := updateWith(ctx, t, r, prior, plan); resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if len(api.puts) != 0 {
		t.Errorf("an update without a change should send nothing: %v", api.puts)
	}
}

func TestDomainResourceRefusesNameserversWithAGroup(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	r := api.resource()

	plan := settingsBase()
	plan.ID, plan.Status = types.StringUnknown(), types.StringUnknown()
	plan.Period, plan.MaxCost = types.Int64Value(1), types.Int64Null()
	plan.NSGroup = types.StringValue("dns-openprovider")
	plan.Nameservers = nameserverList(t, NameserverModel{Name: types.StringValue("ns1.example.net"), IP: types.StringNull(), IP6: types.StringNull()})

	planState := stateOf(ctx, t, r, plan)
	resp := resource.CreateResponse{State: planState}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(planState)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("the create should fail")
	}
	if len(api.posts) != 0 {
		t.Errorf("the provider should send no order: %v", api.posts)
	}
}

func readSettings(ctx context.Context, t *testing.T, r *DomainResource, state DomainModel) DomainModel {
	t.Helper()
	s := stateOf(ctx, t, r, state)
	resp := resource.ReadResponse{State: s}
	r.Read(ctx, resource.ReadRequest{State: s}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	return got
}

func TestDomainResourceReadMapsTheNameservers(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	api.nameservers = `[{"name":"ns2.example.net","ip":"192.0.2.2","seq_nr":2},{"name":"ns1.example.net","ip":"","seq_nr":1}]`
	r := api.resource()

	// A configuration that sets no nameservers does not manage them.
	if got := readSettings(ctx, t, r, settingsBase()); !got.Nameservers.IsNull() {
		t.Errorf("unmanaged nameservers should stay null, got %v", got.Nameservers)
	}

	// With a list in the state they are read in the order of seq_nr, and an
	// empty address is null, as a configuration that leaves it out has it.
	managed := settingsBase()
	managed.Nameservers = nameserverList(t, NameserverModel{Name: types.StringValue("x"), IP: types.StringNull(), IP6: types.StringNull()})
	got := readSettings(ctx, t, r, managed)
	want := nameserverList(t,
		NameserverModel{Name: types.StringValue("ns1.example.net"), IP: types.StringNull(), IP6: types.StringNull()},
		NameserverModel{Name: types.StringValue("ns2.example.net"), IP: types.StringValue("192.0.2.2"), IP6: types.StringNull()},
	)
	if !got.Nameservers.Equal(want) {
		t.Errorf("nameservers: got %v, want %v", got.Nameservers, want)
	}
}

func TestDomainResourceReadMapsWhoisPrivacyAndComments(t *testing.T) {
	ctx := context.Background()
	api := newSettingsAPI(t)
	api.whois, api.allowed, api.comments = true, false, "changed at the panel"
	r := api.resource()

	got := readSettings(ctx, t, r, settingsBase())
	if !got.IsPrivateWhoisEnabled.ValueBool() || got.IsPrivateWhoisAllowed.ValueBool() || got.Comments.ValueString() != "changed at the panel" {
		t.Errorf("state: got %+v", got)
	}
}
