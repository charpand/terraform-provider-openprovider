// Package domains_test contains tests for the domains package.
package domains_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestGetDomain(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	// Replace 123 with an example ID that exists in your OpenAPI examples/mock
	// The Prism mock server will return sample data based on the swagger examples.
	domain, err := domains.Get(apiClient, 123)

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
