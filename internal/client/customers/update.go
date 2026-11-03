// Package customers provides functionality for working with customers.
package customers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/customers/%s", handle)
	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result UpdateCustomerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	return nil
}
