// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// GetDomainResponse represents a response for a single domain.
type GetDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Get retrieves a single domain by ID from the Openprovider API.
//
// Endpoint: GET https://api.openprovider.eu/v1/domains/{id}
func Get(c *client.Client, id int) (*Domain, error) {
	path := fmt.Sprintf("/v1/domains/%d", id)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result GetDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
