// Package domains_test contains tests for the domains package.
package client

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestGetDomain(t *testing.T) {
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

	// Replace 123 with an example ID that exists in your OpenAPI examples/mock
	// The Prism mock server will return sample data based on the swagger examples.
	domain, err := Get(client, 123)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	if domain.Domain.Name == "" {
		t.Errorf("Expected domain name to be populated")
	}

	// Based on the swagger example
	if domain.ID == 123456789 {
		if domain.Domain.Name != "test4" {
			t.Errorf("Expected domain name test4, got %s", domain.Domain.Name)
		}
		if domain.Domain.Extension != "london" {
			t.Errorf("Expected domain extension london, got %s", domain.Domain.Extension)
		}
	}
}
