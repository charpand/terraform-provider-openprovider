package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// priceAPI stands in for the price endpoint, answering every quote with the
// given reseller price and currency.
func priceAPI(t *testing.T, price float64, currency string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/domains/prices" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"is_premium":false,"price":{`+
			`"product":{"currency":%q,"price":%v},`+
			`"reseller":{"currency":%q,"price":%v}`+
			`}}}`, currency, price, currency, price)
	}))
}

func guardSpendClient(t *testing.T, server *httptest.Server) *client.Client {
	t.Helper()
	return client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})
}

func TestGuardSpendAllowsAQuoteAtOrBelowMaxCost(t *testing.T) {
	server := priceAPI(t, 15, "EUR")
	defer server.Close()
	c := guardSpendClient(t, server)

	plan := &DomainModel{MaxCost: types.Int64Value(1500)}
	var diags diag.Diagnostics
	cost, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if !ok || diags.HasError() {
		t.Fatalf("expected the quote to be allowed, got ok=%v diags=%v", ok, diags)
	}
	if cost != 1500 {
		t.Errorf("expected a cost of 1500 cents, got %d", cost)
	}
}

// TestGuardSpendRoundsUpToTheCent covers the `math.Ceil` in `guardSpend`: a
// quote of 14.991 must round up to 1500 cents, not down to 1499, so a bound
// is never passed by a fraction of a cent.
func TestGuardSpendRoundsUpToTheCent(t *testing.T) {
	server := priceAPI(t, 14.991, "EUR")
	defer server.Close()
	c := guardSpendClient(t, server)

	plan := &DomainModel{MaxCost: types.Int64Value(1500)}
	var diags diag.Diagnostics
	cost, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if !ok || diags.HasError() {
		t.Fatalf("expected 14.991 rounded up to 1500 to be allowed at max_cost 1500, got ok=%v diags=%v", ok, diags)
	}
	if cost != 1500 {
		t.Errorf("expected the quote to round up to 1500 cents, got %d", cost)
	}

	// A max_cost one cent below the rounded value must still refuse.
	plan.MaxCost = types.Int64Value(1499)
	diags = nil
	_, ok = guardSpend(c, plan, "example", "com", "create", &diags)
	if ok || !diags.HasError() {
		t.Fatal("expected the rounded-up cost to be refused against a max_cost of 1499")
	}
}

func TestGuardSpendRefusesAQuoteAboveMaxCost(t *testing.T) {
	server := priceAPI(t, 20, "EUR")
	defer server.Close()
	c := guardSpendClient(t, server)

	plan := &DomainModel{MaxCost: types.Int64Value(1500)}
	var diags diag.Diagnostics
	_, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if ok || !diags.HasError() {
		t.Fatal("expected a quote above max_cost to be refused")
	}
}

func TestGuardSpendRefusesAnotherCurrency(t *testing.T) {
	server := priceAPI(t, 15, "USD")
	defer server.Close()
	c := guardSpendClient(t, server)

	// currency left unset, defaults to EUR, so a USD quote must be refused
	// rather than silently converted.
	plan := &DomainModel{MaxCost: types.Int64Value(10000)}
	var diags diag.Diagnostics
	_, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if ok || !diags.HasError() {
		t.Fatal("expected a quote in another currency to be refused")
	}
}

func TestGuardSpendRefusesAZeroQuote(t *testing.T) {
	server := priceAPI(t, 0, "EUR")
	defer server.Close()
	c := guardSpendClient(t, server)

	plan := &DomainModel{MaxCost: types.Int64Value(10000)}
	var diags diag.Diagnostics
	_, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if ok || !diags.HasError() {
		t.Fatal("expected an unquoted (zero) price to be refused, not treated as free")
	}
}

func TestGuardSpendRefusesOnATransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	c := guardSpendClient(t, server)

	plan := &DomainModel{MaxCost: types.Int64Value(10000)}
	var diags diag.Diagnostics
	_, ok := guardSpend(c, plan, "example", "com", "create", &diags)
	if ok || !diags.HasError() {
		t.Fatal("expected a failed quote request to refuse rather than proceed")
	}
}
