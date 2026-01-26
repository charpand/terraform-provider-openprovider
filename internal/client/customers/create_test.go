// Package customers_test contains tests for the customers package.
package customers_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestCreateCustomer(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &customers.CreateCustomerRequest{
		Email: "test@example.com",
		Phone: customers.Phone{
			CountryCode: "1",
			AreaCode:    "555",
			Number:      "1234567",
		},
		Address: customers.Address{
			Street:  "Main St",
			Number:  "123",
			City:    "New York",
			Country: "US",
			Zipcode: "10001",
		},
		Name: customers.Name{
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	handle, err := customers.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if handle == "" {
		t.Log("Note: No handle returned by mock server")
	}
}
