// Package customers provides functionality for working with customers.
package customers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateCustomerRequest represents a request to create a customer.
type CreateCustomerRequest struct {
	CompanyName string  `json:"company_name,omitempty"`
	Email       string  `json:"email"`
	Phone       Phone   `json:"phone"`
	Address     Address `json:"address"`
	Name        Name    `json:"name"`
	Locale      string  `json:"locale,omitempty"`
	Comments    string  `json:"comments,omitempty"`
}

// CreateCustomerResponse represents a response for creating a customer.
type CreateCustomerResponse struct {
	Code int `json:"code"`
	Data struct {
		Handle string `json:"handle"`
	} `json:"data"`
}

// Create creates a new customer via the Openprovider API.
//
// Endpoint: POST https://api.openprovider.eu/v1/customers
func Create(c *client.Client, req *CreateCustomerRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	path := "/v1/customers"
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result CreateCustomerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.Handle, nil
}
