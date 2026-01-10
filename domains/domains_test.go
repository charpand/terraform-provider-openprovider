// Package domains_test contains tests for the domains package.
package domains_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/openprovider-go"
	"github.com/charpand/openprovider-go/domains"
)

type mockTransport struct {
	rt http.RoundTripper
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer dummy")
	req.Header.Set("Prefer", "code=200")
	return t.rt.RoundTrip(req)
}

func TestListDomains(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &mockTransport{rt: http.DefaultTransport},
	}

	config := openprovider.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	client := openprovider.NewClient(config)

	resp, err := domains.List(client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp) == 0 {
		t.Log("Note: No domains returned by mock server (check your swagger examples)")
	}
}
