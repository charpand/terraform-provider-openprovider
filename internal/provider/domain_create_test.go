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
	"github.com/hashicorp/terraform-plugin-framework/path"
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
// transfer has no period, and keeps null. A transfer does not send its
// period, so its quote uses one year, also when it states a period below one.
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
		wantQuoted  string
		wantOrdered any
		wantState   types.Int64
	}{
		"registration, period unknown": {
			authCode:    types.StringNull(),
			period:      types.Int64Unknown(),
			wantPath:    "/v1beta/domains",
			wantQuoted:  "1",
			wantOrdered: float64(1),
			wantState:   types.Int64Value(1),
		},
		"registration, period 2": {
			authCode:    types.StringNull(),
			period:      types.Int64Value(2),
			wantPath:    "/v1beta/domains",
			wantQuoted:  "2",
			wantOrdered: float64(2),
			wantState:   types.Int64Value(2),
		},
		"transfer, period unknown": {
			authCode:    types.StringValue("secret"),
			period:      types.Int64Unknown(),
			wantPath:    "/v1beta/domains/transfer",
			wantQuoted:  "1",
			wantOrdered: nil,
			wantState:   types.Int64Null(),
		},
		"transfer, period 0": {
			authCode:    types.StringValue("secret"),
			period:      types.Int64Value(0),
			wantPath:    "/v1beta/domains/transfer",
			wantQuoted:  "1",
			wantOrdered: nil,
			wantState:   types.Int64Value(0),
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
			if len(api.quoted) != 1 || api.quoted[0] != tc.wantQuoted {
				t.Errorf("quoted periods: got %v, want [%s]", api.quoted, tc.wantQuoted)
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
// below one year for a registration at plan time, so `Create` never orders a
// period other than the one it keeps in state. A period left out stays valid,
// and a transfer, which does not send `period`, accepts any period.
func TestPeriodRejectsBelowOne(t *testing.T) {
	ctx := context.Background()
	r := &DomainResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	period, ok := schemaResp.Schema.Attributes["period"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("period should be an Int64Attribute")
	}

	cases := map[string]struct {
		in       types.Int64
		authCode types.String
		wantErr  bool
	}{
		"negative":              {types.Int64Value(-1), types.StringNull(), true},
		"zero":                  {types.Int64Value(0), types.StringNull(), true},
		"zero, empty auth_code": {types.Int64Value(0), types.StringValue(""), true},
		"zero, unknown auth":    {types.Int64Value(0), types.StringUnknown(), true},
		"one":                   {types.Int64Value(1), types.StringNull(), false},
		"ten":                   {types.Int64Value(10), types.StringNull(), false},
		"null":                  {types.Int64Null(), types.StringNull(), false},
		"unknown":               {types.Int64Unknown(), types.StringNull(), false},
		"zero, transfer":        {types.Int64Value(0), types.StringValue("secret"), false},
		"negative, transfer":    {types.Int64Value(-1), types.StringValue("secret"), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			state := stateOf(ctx, t, r, DomainModel{
				Period:     tc.in,
				AuthCode:   tc.authCode,
				DnssecKeys: types.ListNull(types.ObjectType{AttrTypes: dnssecKeysAttrTypes}),
			})
			req := validator.Int64Request{
				Path:        path.Root("period"),
				ConfigValue: tc.in,
				Config:      tfsdk.Config(state),
			}
			resp := &validator.Int64Response{}
			for _, v := range period.Int64Validators() {
				v.ValidateInt64(ctx, req, resp)
			}
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
