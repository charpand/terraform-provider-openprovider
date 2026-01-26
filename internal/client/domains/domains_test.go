// Package domains_test contains tests for the domains package.
package domains_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListDomains(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	resp, err := domains.List(apiClient)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp) == 0 {
		t.Log("Note: No domains returned by mock server (check your swagger examples)")
	}
}
