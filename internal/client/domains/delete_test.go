// Package domains_test contains tests for the domains package.
package domains_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestDeleteDomain(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	// Delete a test domain
	err := domains.Delete(apiClient, 123)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
