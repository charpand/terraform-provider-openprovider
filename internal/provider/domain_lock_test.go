package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// lockAPI stands in for an account that holds one active domain with a lock.
// An update sets the lock from its body, and the listing shows the lock that
// the account has. It keeps the lock value of each update it receives.
type lockAPI struct {
	*httptest.Server
	locked  bool
	updates []any
}

func newLockAPI(t *testing.T, locked bool) *lockAPI {
	t.Helper()
	api := &lockAPI{locked: locked}
	domain := func() string {
		return fmt.Sprintf(`{"id":1,"status":"ACT","autorenew":"off","owner_handle":"XX000001-NL",`+
			`"is_locked":%t,"domain":{"name":"example","extension":"com"}}`, api.locked)
	}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains/prices":
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"is_premium":false,"price":{`+
				`"product":{"currency":"EUR","price":15},`+
				`"reseller":{"currency":"EUR","price":15}}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1beta/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain() + `}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":{"results":[` + domain() + `],"total":1}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1beta/domains/1":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read update: %v", err)
			}
			var update map[string]any
			if err := json.Unmarshal(body, &update); err != nil {
				t.Errorf("decode update %s: %v", body, err)
			}
			api.updates = append(api.updates, update["is_locked"])
			if locked, ok := update["is_locked"].(bool); ok {
				api.locked = locked
			}
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain() + `}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return api
}

// TestDomainResourceIsLockedIsOptionalAndComputed checks that `is_locked`
// keeps the value at OpenProvider when the configuration does not set it.
func TestDomainResourceIsLockedIsOptionalAndComputed(t *testing.T) {
	ctx := context.Background()
	r := NewDomainResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)

	lockedAttr, ok := resp.Schema.Attributes["is_locked"].(schema.BoolAttribute)
	if !ok {
		t.Fatal("is_locked attribute not found in schema, or not a BoolAttribute")
	}
	if !lockedAttr.Optional || !lockedAttr.Computed {
		t.Error("is_locked should be Optional and Computed")
	}
	if lockedAttr.Default != nil {
		t.Error("is_locked should have no default, so an unset value does not change the lock")
	}
	if len(lockedAttr.PlanModifiers) == 0 {
		t.Error("is_locked should carry a plan modifier so it does not go unknown when not set")
	}
}

// TestDomainResourceCreateSetsTheLock covers a create. The order request
// carries no lock, so a configured lock that the new domain does not have is
// set with an update, and an unset lock takes the value at OpenProvider.
func TestDomainResourceCreateSetsTheLock(t *testing.T) {
	ctx := context.Background()

	noKeys := types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes})
	base := DomainModel{
		ID:          types.StringUnknown(),
		Status:      types.StringUnknown(),
		Domain:      types.StringValue("example.com"),
		OwnerHandle: types.StringValue("XX000001-NL"),
		Autorenew:   types.BoolValue(false),
		DnssecKeys:  noKeys,
		Period:      types.Int64Value(1),
		MaxCost:     types.Int64Value(10000),
		Currency:    types.StringValue("EUR"),
		OnDestroy:   types.StringValue(onDestroyRetain),
	}

	for name, tc := range map[string]struct {
		planned     types.Bool
		wantUpdates []any
		wantState   types.Bool
	}{
		"not set": {
			planned:     types.BoolUnknown(),
			wantUpdates: nil,
			wantState:   types.BoolValue(true),
		},
		"set to the value at OpenProvider": {
			planned:     types.BoolValue(true),
			wantUpdates: nil,
			wantState:   types.BoolValue(true),
		},
		"set to a different value": {
			planned:     types.BoolValue(false),
			wantUpdates: []any{false},
			wantState:   types.BoolValue(false),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := newLockAPI(t, true)
			defer api.Close()
			r := &DomainResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})}

			plan := base
			plan.IsLocked = tc.planned
			got := createDomain(ctx, t, r, plan)

			if fmt.Sprint(api.updates) != fmt.Sprint(tc.wantUpdates) {
				t.Errorf("lock updates: got %v, want %v", api.updates, tc.wantUpdates)
			}
			if !got.IsLocked.Equal(tc.wantState) {
				t.Errorf("is_locked: got %v, want %v", got.IsLocked, tc.wantState)
			}
		})
	}
}

// TestDomainResourceUpdateSetsTheLock covers an update. A planned lock that
// the state does not hold is sent. An unknown planned lock, which comes from
// a prior state that holds no lock, is not sent, and the refresh fills it.
func TestDomainResourceUpdateSetsTheLock(t *testing.T) {
	ctx := context.Background()

	noKeys := types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes})
	base := DomainModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		Status:      types.StringValue("ACT"),
		Autorenew:   types.BoolValue(false),
		OwnerHandle: types.StringValue("XX000001-NL"),
		DnssecKeys:  noKeys,
		OnDestroy:   types.StringValue(onDestroyRetain),
	}

	for name, tc := range map[string]struct {
		prior, planned types.Bool
		wantUpdates    []any
		wantState      types.Bool
	}{
		"unlock": {
			prior:       types.BoolValue(true),
			planned:     types.BoolValue(false),
			wantUpdates: []any{false},
			wantState:   types.BoolValue(false),
		},
		"no change": {
			prior:       types.BoolValue(true),
			planned:     types.BoolValue(true),
			wantUpdates: nil,
			wantState:   types.BoolValue(true),
		},
		"a prior state without a lock": {
			prior:       types.BoolNull(),
			planned:     types.BoolUnknown(),
			wantUpdates: nil,
			wantState:   types.BoolValue(true),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := newLockAPI(t, true)
			defer api.Close()
			r := &DomainResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})}

			prior := base
			prior.IsLocked = tc.prior
			plan := base
			plan.IsLocked = tc.planned
			priorState := stateOf(ctx, t, r, prior)
			resp := resource.UpdateResponse{State: priorState}
			r.Update(ctx, resource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: priorState.Schema, Raw: stateOf(ctx, t, r, plan).Raw},
				State: priorState,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("update: %v", resp.Diagnostics)
			}
			var got DomainModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("get: %v", diags)
			}

			if fmt.Sprint(api.updates) != fmt.Sprint(tc.wantUpdates) {
				t.Errorf("lock updates: got %v, want %v", api.updates, tc.wantUpdates)
			}
			if !got.IsLocked.Equal(tc.wantState) {
				t.Errorf("is_locked: got %v, want %v", got.IsLocked, tc.wantState)
			}
		})
	}
}
