package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func readPrice(t *testing.T, answer string, config DomainPriceDataSourceModel) (DomainPriceDataSourceModel, []url.Values, bool) {
	t.Helper()
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/domains/prices" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		queries = append(queries, r.URL.Query())
		_, _ = fmt.Fprint(w, answer)
	}))
	defer server.Close()

	d := &DomainPriceDataSource{client: client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})}
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	config.Price, config.Currency, config.ProductPrice = types.Float64Unknown(), types.StringUnknown(), types.Float64Unknown()
	config.ResellerPrice, config.IsPremium, config.ID = types.Float64Unknown(), types.BoolUnknown(), types.StringUnknown()
	staged := tfsdk.State{Schema: schemaResp.Schema}
	if diags := staged.Set(ctx, config); diags.HasError() {
		t.Fatalf("setting config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: staged.Raw}}, resp)
	var state DomainPriceDataSourceModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("reading state: %v", diags)
		}
	}
	return state, queries, resp.Diagnostics.HasError()
}

const priceAnswer = `{"code":0,"data":{"is_premium":false,"price":{` +
	`"product":{"currency":"EUR","price":12.5},"reseller":{"currency":"EUR","price":9.75}}}}`

func TestDomainPriceDefaultsToARegistration(t *testing.T) {
	state, queries, errored := readPrice(t, priceAnswer, DomainPriceDataSourceModel{
		Domain: types.StringValue("example.com"), Operation: types.StringNull(), Period: types.Int64Null(),
	})
	if errored {
		t.Fatal("the read should not fail")
	}
	q := queries[0]
	if q.Get("domain.name") != "example" || q.Get("domain.extension") != "com" || q.Get("operation") != "create" || q.Get("period") != "1" {
		t.Errorf("query: got %v", q)
	}
	if state.Price.ValueFloat64() != 9.75 || state.Currency.ValueString() != "EUR" ||
		state.ProductPrice.ValueFloat64() != 12.5 || state.ResellerPrice.ValueFloat64() != 9.75 {
		t.Errorf("state: got %+v", state)
	}
	if state.ID.ValueString() != "example.com/create/1" {
		t.Errorf("id: got %v", state.ID)
	}
}

func TestDomainPriceWithoutAResellerPriceChargesTheRetailPrice(t *testing.T) {
	state, _, errored := readPrice(t, `{"code":0,"data":{"price":{"product":{"currency":"EUR","price":12.5},"reseller":{"currency":"","price":0}}}}`,
		DomainPriceDataSourceModel{Domain: types.StringValue("example.com"), Operation: types.StringNull(), Period: types.Int64Null()})
	if errored {
		t.Fatal("the read should not fail")
	}
	if state.Price.ValueFloat64() != 12.5 || state.Currency.ValueString() != "EUR" {
		t.Errorf("a missing reseller price should charge the retail price, got %+v", state)
	}
}

func TestDomainPriceOfATransferIsForOneYear(t *testing.T) {
	_, queries, errored := readPrice(t, priceAnswer, DomainPriceDataSourceModel{
		Domain: types.StringValue("example.com"), Operation: types.StringValue("transfer"), Period: types.Int64Value(5),
	})
	if errored {
		t.Fatal("the read should not fail")
	}
	if queries[0].Get("operation") != "transfer" || queries[0].Get("period") != "1" {
		t.Errorf("query: got %v", queries[0])
	}
}

func TestDomainPriceRejectsWrongInput(t *testing.T) {
	for name, config := range map[string]DomainPriceDataSourceModel{
		"no extension":      {Domain: types.StringValue("example"), Operation: types.StringNull(), Period: types.Int64Null()},
		"unknown operation": {Domain: types.StringValue("example.com"), Operation: types.StringValue("delete"), Period: types.Int64Null()},
		"period too long":   {Domain: types.StringValue("example.com"), Operation: types.StringNull(), Period: types.Int64Value(11)},
		"period zero":       {Domain: types.StringValue("example.com"), Operation: types.StringNull(), Period: types.Int64Value(0)},
	} {
		t.Run(name, func(t *testing.T) {
			_, queries, errored := readPrice(t, priceAnswer, config)
			if !errored {
				t.Fatal("the read should fail")
			}
			if len(queries) != 0 {
				t.Errorf("the provider should send no request: %v", queries)
			}
		})
	}
}

func TestDomainPriceRefused(t *testing.T) {
	_, _, errored := readPrice(t, `{"code":300,"desc":"Your domain request contains an empty domain name"}`, DomainPriceDataSourceModel{
		Domain: types.StringValue("example.com"), Operation: types.StringNull(), Period: types.Int64Null(),
	})
	if !errored {
		t.Fatal("the read should fail")
	}
}
