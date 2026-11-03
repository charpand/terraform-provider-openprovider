// Package customers provides functionality for working with customers.
package customers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Address represents a customer address.
type Address struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Number  string `json:"number,omitempty"`
	State   string `json:"state,omitempty"`
	Street  string `json:"street"`
	Suffix  string `json:"suffix,omitempty"`
	Zipcode string `json:"zipcode,omitempty"`
}

// Name represents a customer name.
type Name struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Initials  string `json:"initials,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

// Phone represents a customer phone.
type Phone struct {
	AreaCode    string `json:"area_code"`
	CountryCode string `json:"country_code"`
	Number      string `json:"subscriber_number"`
}

// Customer represents a customer entity.
type Customer struct {
	ID          int     `json:"id"`
	Handle      string  `json:"handle"`
	CompanyName string  `json:"company_name,omitempty"`
	Email       string  `json:"email"`
	Phone       Phone   `json:"phone"`
	Address     Address `json:"address"`
	Name        Name    `json:"name"`
	Locale      string  `json:"locale,omitempty"`
	Comments    string  `json:"comments,omitempty"`
}

// ListCustomersResponse represents a response from the customers listing endpoint.
type ListCustomersResponse struct {
	Code int `json:"code"`
	Data struct {
		Results []Customer `json:"results"`
		Total   int        `json:"total"`
	} `json:"data"`
}

// List retrieves a list of customers from the Openprovider API.
func List(c *client.Client) ([]Customer, error) {
	path := "/v1/customers"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var results ListCustomersResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results.Data.Results, nil
}
