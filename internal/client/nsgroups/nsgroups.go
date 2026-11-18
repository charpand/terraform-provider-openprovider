// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Nameserver represents a nameserver in a nameserver group.
type Nameserver struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"name"`
	IP    string `json:"ip,omitempty"`
	IP6   string `json:"ip6,omitempty"`
	SeqNr int    `json:"seq_nr,omitempty"`
}

// NSGroup represents a nameserver group entity.
type NSGroup struct {
	ID          int          `json:"id"`
	Name        string       `json:"ns_group"`
	Nameservers []Nameserver `json:"name_servers"`
	DomainCount int          `json:"domain_count,omitempty"`
	NsCount     int          `json:"ns_count,omitempty"`
	CreatedAt   string       `json:"created_at,omitempty"`
	UpdatedAt   string       `json:"updated_at,omitempty"`
}

// ListNSGroupsResponse represents a response from the NS groups listing endpoint.
type ListNSGroupsResponse struct {
	Code int `json:"code"`
	Data struct {
		Results []NSGroup `json:"results"`
		Total   int       `json:"total"`
	} `json:"data"`
}

// GetNSGroupResponse represents a response for getting a single NS group.
type GetNSGroupResponse struct {
	Code int     `json:"code"`
	Data NSGroup `json:"data"`
}

// List retrieves a list of nameserver groups from the Openprovider API.
func List(c *client.Client) ([]NSGroup, error) {
	var results ListNSGroupsResponse
	if err := c.JSON("GET", "/v1/dns/nameservers/groups", nil, &results); err != nil {
		return nil, err
	}
	return results.Data.Results, nil
}

// Get retrieves a specific nameserver group by name from the Openprovider API.
// The ns_group parameter is the group name. It returns (nil, nil) for a group
// that no longer exists, so a caller can tell "gone" apart from a failed
// request.
func Get(c *client.Client, name string) (*NSGroup, error) {
	var result GetNSGroupResponse
	err := c.JSON("GET", fmt.Sprintf("/v1/dns/nameservers/groups/%s", name), nil, &result)
	if client.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
