// Package customers_test contains tests for the customers package.
package customers_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestUpdateCustomer(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &customers.UpdateCustomerRequest{
		Email: "updated@example.com",
	}

	err := customers.Update(apiClient, "XX123456-XX", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
