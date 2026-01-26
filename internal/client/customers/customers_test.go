// Package customers_test contains tests for the customers package.
package customers_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListCustomers(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	customerList, err := customers.List(apiClient)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if customerList == nil {
		t.Log("Note: No customers returned by mock server")
		return
	}

	t.Logf("Returned %d customers", len(customerList))
}
