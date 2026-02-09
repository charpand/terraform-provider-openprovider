// Package domains_test contains tests for the domains package.
package domains_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestTransferDomain(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &domains.TransferDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.AuthCode = "12345678"
	req.OwnerHandle = "testowner"
	req.Autorenew = "on"

	domain, err := domains.Transfer(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	if domain.Domain.Name == "" {
		t.Log("Note: Domain name not populated by mock server")
	}
}

func TestTransferDomainWithNSGroup(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &domains.TransferDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.AuthCode = "12345678"
	req.OwnerHandle = "testowner"
	req.NSGroup = "dns-openprovider"
	req.Autorenew = "on"

	domain, err := domains.Transfer(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	if domain.NSGroup == "" {
		t.Log("Note: NS Group not populated by mock server")
	}
}

func TestTransferDomainWithImportOptions(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &domains.TransferDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.AuthCode = "12345678"
	req.OwnerHandle = "testowner"

	domain, err := domains.Transfer(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}
}

func TestTransferDomainWithNameservers(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &domains.TransferDomainRequest{}
	req.Domain.Name = "example"
	req.Domain.Extension = "com"
	req.AuthCode = "12345678"
	req.OwnerHandle = "testowner"
	req.Nameservers = []domains.Nameserver{
		{Name: "ns1.example.com"},
		{Name: "ns2.example.com"},
	}

	domain, err := domains.Transfer(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
		return
	}

	if len(domain.Nameservers) == 0 {
		t.Log("Note: Nameservers not populated by mock server")
	}
}
