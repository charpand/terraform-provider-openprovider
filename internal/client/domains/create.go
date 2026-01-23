// Package domains provides functionality for working with domains.
package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateDomainRequest represents a request to create a domain.
type CreateDomainRequest struct {
	Domain struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
	OwnerHandle   string       `json:"owner_handle"`
	AdminHandle   string       `json:"admin_handle,omitempty"`
	TechHandle    string       `json:"tech_handle,omitempty"`
	BillingHandle string       `json:"billing_handle,omitempty"`
	Period        int          `json:"period,omitempty"`
	Autorenew     string       `json:"autorenew,omitempty"`
	Nameservers   []Nameserver `json:"name_servers,omitempty"`
}

// CreateDomainResponse represents a response for creating a domain.
type CreateDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Create creates a new domain via the Openprovider API.
//
// Endpoint: POST https://api.eu/v1beta/domains
func Create(c *client.Client, req *CreateDomainRequest) (*Domain, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/domains"
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result CreateDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
