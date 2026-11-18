// Package customers provides functionality for working with customers.
package customers

import (
	"fmt"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// GetCustomerResponse represents a response for getting a customer.
type GetCustomerResponse struct {
	Code int `json:"code"`
	Data struct {
		Handle      string  `json:"handle"`
		ID          int     `json:"id"`
		CompanyName string  `json:"company_name,omitempty"`
		Email       string  `json:"email"`
		Phone       Phone   `json:"phone"`
		Address     Address `json:"address"`
		Name        Name    `json:"name"`
		Locale      string  `json:"locale,omitempty"`
		Comments    string  `json:"comments,omitempty"`
	} `json:"data"`
}

// Get retrieves a customer by handle from the Openprovider API.
//
// Endpoint: GET https://api.openprovider.eu/v1/customers/{handle}
// Returns (nil, nil) if the customer is not found (404).
func Get(c *client.Client, handle string) (*Customer, error) {
	var result GetCustomerResponse
	err := c.JSON("GET", fmt.Sprintf("/v1/customers/%s", handle), nil, &result)
	if client.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Map the response data to a Customer struct
	return &Customer{
		ID:          result.Data.ID,
		Handle:      result.Data.Handle,
		CompanyName: result.Data.CompanyName,
		Email:       result.Data.Email,
		Phone:       result.Data.Phone,
		Address:     result.Data.Address,
		Name:        result.Data.Name,
		Locale:      result.Data.Locale,
		Comments:    result.Data.Comments,
	}, nil
}

// Exists reports whether the account holds a customer with the handle.
//
// A deleted customer does not exist. Exists decodes only the fields that it
// needs, so that the data of the customer does not reach the caller.
//
// Endpoint: GET https://api.openprovider.eu/v1/customers/{handle}
func Exists(c *client.Client, handle string) (bool, error) {
	var result struct {
		Data struct {
			Handle    string `json:"handle"`
			IsDeleted bool   `json:"is_deleted"`
		} `json:"data"`
	}
	err := c.JSON("GET", fmt.Sprintf("/v1/customers/%s", url.PathEscape(handle)), nil, &result)
	if client.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if result.Data.IsDeleted {
		return false, nil
	}
	if result.Data.Handle == "" {
		return false, fmt.Errorf("the answer for customer %q has no handle", handle)
	}

	return true, nil
}
