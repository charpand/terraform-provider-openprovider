// Package domains provides functionality for working with domains.
package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// TransferDomainRequest represents a request to transfer a domain.
type TransferDomainRequest struct {
	Domain struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
	AuthCode                      string       `json:"auth_code"`
	OwnerHandle                   string       `json:"owner_handle"`
	AdminHandle                   string       `json:"admin_handle,omitempty"`
	TechHandle                    string       `json:"tech_handle,omitempty"`
	BillingHandle                 string       `json:"billing_handle,omitempty"`
	Autorenew                     string       `json:"autorenew,omitempty"`
	NSGroup                       string       `json:"ns_group,omitempty"`
	Nameservers                   []Nameserver `json:"name_servers,omitempty"`
	ImportContactsFromRegistry    bool         `json:"import_contacts_from_registry,omitempty"`
	ImportNameserversFromRegistry bool         `json:"import_nameservers_from_registry,omitempty"`
	IsPrivateWhoisEnabled         bool         `json:"is_private_whois_enabled,omitempty"`
}

// TransferDomainResponse represents a response for transferring a domain.
type TransferDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Transfer initiates a domain transfer via the Openprovider API.
//
// Endpoint: POST https://api.openprovider.eu/v1beta/domains/transfer
func Transfer(c *client.Client, req *TransferDomainRequest) (*Domain, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/domains/transfer"
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status code %d", resp.StatusCode)
	}

	var result TransferDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
