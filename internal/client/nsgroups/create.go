// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateNSGroupRequest represents a request to create a nameserver group.
type CreateNSGroupRequest struct {
	Name        string       `json:"ns_group"`
	Nameservers []Nameserver `json:"name_servers"`
}

// CreateNSGroupResponse represents a response for creating a nameserver group.
type CreateNSGroupResponse struct {
	Code int     `json:"code"`
	Data NSGroup `json:"data"`
}

// Create creates a new nameserver group via the Openprovider API.
//
// Endpoint: POST https://api.openprovider.eu/v1/dns/nameservers/groups
func Create(c *client.Client, req *CreateNSGroupRequest) (*NSGroup, error) {
	var result CreateNSGroupResponse
	if err := c.JSON("POST", "/v1/dns/nameservers/groups", req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
