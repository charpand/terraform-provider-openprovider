// Package domains provides functionality for working with domains.
package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// UpdateDomainRequest represents a request to update a domain.
type UpdateDomainRequest struct {
	AdminHandle     string       `json:"admin_handle,omitempty"`
	TechHandle      string       `json:"tech_handle,omitempty"`
	BillingHandle   string       `json:"billing_handle,omitempty"`
	Autorenew       string       `json:"autorenew,omitempty"`
	IsLocked        *bool        `json:"is_locked,omitempty"`
	Nameservers     []Nameserver `json:"name_servers,omitempty"`
	NSGroup         string       `json:"ns_group,omitempty"`
	DnssecKeys      []DnssecKey  `json:"dnssec_keys,omitempty"`
	IsDnssecEnabled *bool        `json:"is_dnssec_enabled,omitempty"`
}

// UpdateDomainResponse represents a response for updating a domain.
type UpdateDomainResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data Domain `json:"data"`
}

// Update updates an existing domain by ID via the Openprovider API.
//
// Endpoint: PUT https://api.openprovider.eu/v1beta/domains/{id}
func Update(c *client.Client, id int, req *UpdateDomainRequest) (*Domain, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/domains/%d", id)
	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result UpdateDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// The API can refuse an update with a non-zero code in a 2xx response.
	if result.Code != 0 {
		return nil, fmt.Errorf("update operation failed with code %d: %s", result.Code, result.Desc)
	}

	return &result.Data, nil
}
