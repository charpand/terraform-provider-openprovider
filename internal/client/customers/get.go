// Package customers provides functionality for working with customers.
package customers

import (
	"encoding/json"
	"fmt"
	"net/http"

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
// Endpoint: GET https://api.openprovider.eu/v1beta/customers/{handle}
// Returns (nil, nil) if the customer is not found (404).
func Get(c *client.Client, handle string) (*Customer, error) {
	path := fmt.Sprintf("/v1beta/customers/%s", handle)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		// Check if it's a 404 error
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result GetCustomerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Map the response data to a Customer struct
	customer := &Customer{
		ID:          result.Data.ID,
		Handle:      result.Data.Handle,
		CompanyName: result.Data.CompanyName,
		Email:       result.Data.Email,
		Phone:       result.Data.Phone,
		Address:     result.Data.Address,
		Name:        result.Data.Name,
		Locale:      result.Data.Locale,
		Comments:    result.Data.Comments,
	}

	return customer, nil
}
