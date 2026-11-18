// Package domains provides functionality for working with domains.
package domains

import (
	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateDomainRequest represents a request to create a domain.
type CreateDomainRequest struct {
	Domain struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
	OwnerHandle     string            `json:"owner_handle"`
	AdminHandle     string            `json:"admin_handle,omitempty"`
	TechHandle      string            `json:"tech_handle,omitempty"`
	BillingHandle   string            `json:"billing_handle,omitempty"`
	Period          int               `json:"period,omitempty"`
	Autorenew       string            `json:"autorenew,omitempty"`
	Nameservers     []Nameserver      `json:"name_servers,omitempty"`
	NSGroup         string            `json:"ns_group,omitempty"`
	DnssecKeys      []DnssecKey       `json:"dnssec_keys,omitempty"`
	IsDnssecEnabled *bool             `json:"is_dnssec_enabled,omitempty"`
	AdditionalData  map[string]string `json:"additional_data,omitempty"`
	// IsPrivateWhoisEnabled hides the owner data in the WHOIS. A pointer, so
	// that nil sends nothing and the registry keeps its default.
	IsPrivateWhoisEnabled *bool  `json:"is_private_whois_enabled,omitempty"`
	Comments              string `json:"comments,omitempty"`
}

// CreateDomainResponse represents a response for creating a domain.
type CreateDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Create creates a new domain via the Openprovider API.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains
func Create(c *client.Client, req *CreateDomainRequest) (*Domain, error) {
	var result CreateDomainResponse
	if err := c.JSON("POST", "/v1/domains", req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
