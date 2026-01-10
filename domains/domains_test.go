// Package domains_test contains tests for the domains package.
package domains_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/openprovider-go"
	"github.com/charpand/openprovider-go/domains"
	"github.com/charpand/openprovider-go/internal/testutils"
)

func TestListDomains(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
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
