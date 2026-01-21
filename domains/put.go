// Package domains provides functionality for working with domains.
package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/charpand/openprovider-go"
)

// UpdateDomainRequest represents a request to update a domain.
type UpdateDomainRequest struct {
	AdminHandle   string `json:"admin_handle,omitempty"`
	TechHandle    string `json:"tech_handle,omitempty"`
	BillingHandle string `json:"billing_handle,omitempty"`
	Autorenew     string `json:"autorenew,omitempty"`
	IsLocked      *bool  `json:"is_locked,omitempty"`
}

// UpdateDomainResponse represents a response for updating a domain.
type UpdateDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Update updates an existing domain by ID via the Openprovider API.
//
// Endpoint: PUT https://api.openprovider.eu/v1beta/domains/{id}
func Update(c *openprovider.Client, id int, req *UpdateDomainRequest) (*Domain, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1beta/domains/%d", c.BaseURL, id), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	var result UpdateDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
