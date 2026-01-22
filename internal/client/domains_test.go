// Package domains_test contains tests for the domains package.
package client

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListDomains(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	client := NewClient(config)

	resp, err := List(client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp) == 0 {
		t.Log("Note: No domains returned by mock server (check your swagger examples)")
	}
}
