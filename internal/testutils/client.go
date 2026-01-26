// Package testutils provides helpers for testing the openprovider client.
package testutils

import (
	"net/http"
	"os"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// SetupTestClient creates a test client with default configuration.
func SetupTestClient() *client.Client {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	return client.NewClient(config)
}
