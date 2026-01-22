// Package domains_test contains tests for the domains package.
package client

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestDeleteDomain(t *testing.T) {
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

	// Delete a test domain
	err := Delete(client, 123)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
