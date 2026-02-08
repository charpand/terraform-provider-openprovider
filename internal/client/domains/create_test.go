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
	apiClient := testutils.SetupTestClient()

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
	apiClient := testutils.SetupTestClient()

	// Create a test domain request with nameservers
	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1
	req.Nameservers = []domains.Nameserver{
		{Name: "ns1.example.com"},
		{Name: "ns2.example.com"},
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

func TestCreateDomainWithDSRecords(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	// Create a test domain request with DS records
	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1
	req.DnssecKeys = []domains.DnssecKey{
		{
			Alg:      8,
			Flags:    257,
			Protocol: 3,
			PubKey:   "AwEAAaz/tAm8yTn4Mfeh5eyI96WSVexTBAvkMgJzkKTOiW1vkIbzxeF3+/4RgWOq7HrxRixHlFlExOLAJr5emLvN7SWXgnLh4+B5xQlNVz8Og8kvArMtNROxVQuCaSnIDdD5LKyWbRd2n9WGe2R8PzgCmr3EgVLrjyBxWezF0jLHwVN8efS3rCj/EWgvIWgb9tarpVUDK/b58Da+sqqls3eNbuv7pr+eoZG+SrDK6nWeL3c6H5Apxz7LjVc1uTIdsIXxuOLYA4/ilBmSVIzuDWfdRUfhHdY6+cn8HFRm+2hM8AnXGXws9555KrUB5qihylGa8subX2Nn6UwNR1AkUTV74bU=",
		},
	}

	domain, err := domains.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	// Optional: check if DS records are populated (not a hard failure)
	if len(domain.DnssecKeys) == 0 {
		t.Log("Note: DS records not populated by mock server")
	}
}

func TestCreateDomainWithError(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.ErrorMockTransport{
			RT:         http.DefaultTransport,
			StatusCode: http.StatusInternalServerError,
		},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	apiClient := client.NewClient(config)

	req := &domains.CreateDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.OwnerHandle = "testowner"
	req.Period = 1

	domain, err := domains.Create(apiClient, req)

	if err == nil {
		t.Fatal("Expected error for 500 status code, got nil")
	}

	if domain != nil {
		t.Errorf("Expected nil domain on error, got %v", domain)
	}

	// Verify error message contains status information
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}
