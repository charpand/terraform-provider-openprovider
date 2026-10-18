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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// orderAPI stands in for what `Create` asks of the API: the price endpoint,
// the registration and transfer endpoints, and the listing it reads the
// domain back from. It keeps the body of the order it receives and the
// period of each quote, so a test can check what was sent.
type orderAPI struct {
	*httptest.Server
	order   map[string]any
	quoted  []string
	ordered string
}

func newOrderAPI(t *testing.T, name, extension string) *orderAPI {
	t.Helper()
	api := &orderAPI{}
	domain := `{"id":1,"status":"ACT","autorenew":"off","owner_handle":"XX000001-NL",` +
		`"domain":{"name":"` + name + `","extension":"` + extension + `"}}`
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains/prices":
			api.quoted = append(api.quoted, r.URL.Query().Get("period"))
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"is_premium":false,"price":{`+
				`"product":{"currency":"EUR","price":15},`+
				`"reseller":{"currency":"EUR","price":15}}}}`)
		case r.Method == http.MethodPost && (r.URL.Path == "/v1beta/domains" || r.URL.Path == "/v1beta/domains/transfer"):
			api.ordered = r.URL.Path
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read order: %v", err)
			}
			if err := json.Unmarshal(body, &api.order); err != nil {
				t.Errorf("decode order %s: %v", body, err)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":` + domain + `}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta/domains":
			_, _ = w.Write([]byte(`{"code":0,"data":{"results":[` + domain + `],"total":1}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return api
}

// createDomain drives `Create` with the plan and returns the state it sets.
func createDomain(ctx context.Context, t *testing.T, r *DomainResource, plan DomainModel) DomainModel {
	t.Helper()
	planState := stateOf(ctx, t, r, plan)
	resp := resource.CreateResponse{State: planState}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(planState)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	var got DomainModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %v", diags)
	}
	return got
}

// TestDomainResourceCreateResolvesThePeriod covers a create whose plan
// holds `period` unknown, which is what a configuration that leaves it out
// plans: the attribute is Optional and Computed, and `UseStateForUnknown`
// has no prior state to use. The applied state must hold a known value, or
// the framework fails the apply after the domain was ordered. A
// registration orders the period it was quoted at and keeps it in state; a
// transfer has no period, and keeps null.
func TestDomainResourceCreateResolvesThePeriod(t *testing.T) {
	ctx := context.Background()

	noKeys := types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes})
	base := DomainModel{
		Domain:      types.StringValue("example.com"),
		OwnerHandle: types.StringValue("XX000001-NL"),
		Autorenew:   types.BoolValue(false),
		DnssecKeys:  noKeys,
		MaxCost:     types.Int64Value(10000),
		Currency:    types.StringValue("EUR"),
		OnDestroy:   types.StringValue(onDestroyRetain),
	}

	for name, tc := range map[string]struct {
		authCode    types.String
		period      types.Int64
		wantPath    string
		wantOrdered any
		wantState   types.Int64
	}{
		"registration, period unknown": {
			authCode:    types.StringNull(),
			period:      types.Int64Unknown(),
			wantPath:    "/v1beta/domains",
			wantOrdered: float64(1),
			wantState:   types.Int64Value(1),
		},
		"registration, period 2": {
			authCode:    types.StringNull(),
			period:      types.Int64Value(2),
			wantPath:    "/v1beta/domains",
			wantOrdered: float64(2),
			wantState:   types.Int64Value(2),
		},
		"transfer, period unknown": {
			authCode:    types.StringValue("secret"),
			period:      types.Int64Unknown(),
			wantPath:    "/v1beta/domains/transfer",
			wantOrdered: nil,
			wantState:   types.Int64Null(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := newOrderAPI(t, "example", "com")
			defer api.Close()
			r := &DomainResource{client: client.NewClient(client.Config{BaseURL: api.URL, Token: "test"})}

			plan := base
			plan.ID = types.StringUnknown()
			plan.Status = types.StringUnknown()
			plan.AuthCode = tc.authCode
			plan.Period = tc.period
			got := createDomain(ctx, t, r, plan)

			if api.ordered != tc.wantPath {
				t.Fatalf("order sent to %q, want %q", api.ordered, tc.wantPath)
			}
			if ordered := api.order["period"]; ordered != tc.wantOrdered {
				t.Errorf("ordered period: got %v, want %v", ordered, tc.wantOrdered)
			}
			if tc.wantOrdered != nil {
				want := fmt.Sprint(tc.wantOrdered)
				if len(api.quoted) != 1 || api.quoted[0] != want {
					t.Errorf("quoted periods: got %v, want [%s]", api.quoted, want)
				}
			}
			if got.Period.IsUnknown() {
				t.Error("period should not be unknown after apply")
			}
			if !got.Period.Equal(tc.wantState) {
				t.Errorf("period: got %v, want %v", got.Period, tc.wantState)
			}
		})
	}
}

// TestPeriodRejectsBelowOne checks that the schema refuses a stated period
// below one year at plan time, so `Create` never orders a period other than
// the one it keeps in state. A period left out stays valid.
func TestPeriodRejectsBelowOne(t *testing.T) {
	schemaResp := &resource.SchemaResponse{}
	NewDomainResource().Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	period, ok := schemaResp.Schema.Attributes["period"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("period should be an Int64Attribute")
	}

	cases := map[string]struct {
		in      types.Int64
		wantErr bool
	}{
		"negative": {types.Int64Value(-1), true},
		"zero":     {types.Int64Value(0), true},
		"one":      {types.Int64Value(1), false},
		"ten":      {types.Int64Value(10), false},
		"null":     {types.Int64Null(), false},
		"unknown":  {types.Int64Unknown(), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.Int64Response{}
			for _, v := range period.Int64Validators() {
				v.ValidateInt64(context.Background(), validator.Int64Request{ConfigValue: tc.in}, resp)
			}
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
