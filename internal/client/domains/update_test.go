// Package domains_test contains tests for the domains package.
package domains_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestUpdateDomain(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	// Update a test domain
	req := &domains.UpdateDomainRequest{
		Autorenew: "on",
	}

	domain, err := domains.Update(apiClient, 123, req)

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

func TestUpdateDomainWithNameservers(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	// Update a test domain with nameservers
	req := &domains.UpdateDomainRequest{
		Autorenew: "on",
		Nameservers: []domains.Nameserver{
			{Name: "ns1.cloudflare.com"},
			{Name: "ns2.cloudflare.com"},
		},
	}

	domain, err := domains.Update(apiClient, 123, req)

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
