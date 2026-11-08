// Package tlds provides functionality for reading the extensions that Openprovider offers.
package tlds

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// TLD holds the contact roles of an extension. A role that the API does not
// report is nil, so that it is not the same as a role that is not supported.
type TLD struct {
	Name                    string `json:"name"`
	OwnerHandleSupported    *bool  `json:"owner_handle_supported"`
	AdminHandleSupported    *bool  `json:"admin_handle_supported"`
	TechHandleSupported     *bool  `json:"tech_handle_supported"`
	BillingHandleSupported  *bool  `json:"billing_handle_supported"`
	ResellerHandleSupported *bool  `json:"reseller_handle_supported"`
}

// GetTLDResponse represents a response for a single extension.
type GetTLDResponse struct {
	Code int `json:"code"`
	Data TLD `json:"data"`
}

// Get retrieves an extension by name, without the dot (e.g., `com`).
//
// Endpoint: GET https://api.openprovider.eu/v1/tlds/{name}
func Get(c *client.Client, name string) (*TLD, error) {
	path := fmt.Sprintf("/v1/tlds/%s", url.PathEscape(name))
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

	var result GetTLDResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Data.Name == "" {
		return nil, fmt.Errorf("the answer for extension %q has no name", name)
	}

	return &result.Data, nil
}
