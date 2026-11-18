// Package domains provides functionality for working with domains.
package domains

import (
	"fmt"

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
	var result GetDomainResponse
	if err := c.JSON("GET", fmt.Sprintf("/v1/domains/%d", id), nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
