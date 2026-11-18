// Package customers provides functionality for working with customers.
package customers

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// UpdateCustomerRequest represents a request to update a customer.
type UpdateCustomerRequest struct {
	CompanyName string   `json:"company_name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Phone       *Phone   `json:"phone,omitempty"`
	Address     *Address `json:"address,omitempty"`
	Name        *Name    `json:"name,omitempty"`
	Locale      string   `json:"locale,omitempty"`
	Comments    string   `json:"comments,omitempty"`
}

// UpdateCustomerResponse represents a response for updating a customer.
type UpdateCustomerResponse struct {
	Code int `json:"code"`
	Data struct {
		Handle string `json:"handle"`
	} `json:"data"`
}

// Update updates an existing customer via the Openprovider API.
//
// Endpoint: PUT https://api.openprovider.eu/v1/customers/{handle}
func Update(c *client.Client, handle string, req *UpdateCustomerRequest) error {
	var result UpdateCustomerResponse
	return c.JSON("PUT", fmt.Sprintf("/v1/customers/%s", handle), req, &result)
}
