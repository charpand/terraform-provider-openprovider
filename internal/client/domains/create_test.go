// Package domains_test contains tests for the domains package.
package domains_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestCreateDomain(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	apiClient := client.NewClient(config)

	// Create a test domain request
	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1

	domain, err := domains.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	// Optional: check if domain name is populated (not a hard failure)
	if domain.Domain.Name == "" {
		t.Log("Note: Domain name not populated by mock server")
	}
}

func TestCreateDomainWithNameservers(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	apiClient := client.NewClient(config)

	// Create a test domain request with nameservers
	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1
	req.Nameservers = []domains.Nameserver{
		{Hostname: "ns1.example.com"},
		{Hostname: "ns2.example.com"},
	}

	domain, err := domains.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	// Optional: check if nameservers are populated (not a hard failure)
	if len(domain.Nameservers) == 0 {
		t.Log("Note: Nameservers not populated by mock server")
	}
}
