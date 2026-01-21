// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/charpand/openprovider-go"
)

// GetDomainResponse represents a response for a single domain.
type GetDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Get retrieves a single domain by ID from the Openprovider API.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/domains/{id}
func Get(c *openprovider.Client, id int) (*Domain, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/domains/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	var result GetDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
