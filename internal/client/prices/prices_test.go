package prices_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/prices"
)

func TestCreateReadsTheQuote(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"is_premium":false,"price":{` +
			`"product":{"currency":"EUR","price":15},` +
			`"reseller":{"currency":"EUR","price":12}}}}`))
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	quote, err := prices.Create(c, "example", "com", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if quote.Price.Reseller.Price != 12 || quote.Price.Reseller.Currency != "EUR" {
		t.Errorf("expected the reseller quote to be read, got %+v", quote.Price.Reseller)
	}
	if charge := quote.Charge(); charge.Price != 12 {
		t.Errorf("expected Charge to prefer the reseller price, got %+v", charge)
	}

	want := "domain.extension=com&domain.name=example&operation=create&period=2"
	if gotQuery != want {
		t.Errorf("expected the request query %q, got %q", want, gotQuery)
	}
}

func TestCreateFallsBackToProductPriceForANonMember(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"is_premium":false,"price":{` +
			`"product":{"currency":"EUR","price":15},` +
			`"reseller":{"currency":"","price":0}}}}`))
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	quote, err := prices.Create(c, "example", "com", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if charge := quote.Charge(); charge.Price != 15 || charge.Currency != "EUR" {
		t.Errorf("expected a zero reseller price to fall back to the product price, got %+v", charge)
	}
}

func TestCreateReportsAnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":300,"desc":"Your domain request contains an empty domain name"}`))
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	if _, err := prices.Create(c, "example", "com", 1); err == nil {
		t.Fatal("expected a non-zero code in the envelope to be an error")
	}
}

func TestCreateReportsAFailedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	if _, err := prices.Create(c, "example", "com", 1); err == nil {
		t.Fatal("expected a failed HTTP request (non-2xx) to be reported as an error")
	}
}
