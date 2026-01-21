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

func TestCreateDomain(t *testing.T) {
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

	// Create a test domain request
	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1

	domain, err := domains.Create(client, req)

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
}
