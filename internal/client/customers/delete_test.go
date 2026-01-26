// Package customers_test contains tests for the customers package.
package customers_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestDeleteCustomer(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	err := customers.Delete(apiClient, "XX123456-XX")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
