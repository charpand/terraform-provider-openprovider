// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Nameserver represents a nameserver in a nameserver group.
type Nameserver struct {
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
	IP6  string `json:"ip6,omitempty"`
}

// NSGroup represents a nameserver group entity.
type NSGroup struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Nameservers []Nameserver `json:"name_servers"`
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
	path := "/v1beta/dns/nameservers/groups"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var results ListNSGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results.Data.Results, nil
}

// Get retrieves a specific nameserver group by ID from the Openprovider API.
func Get(c *client.Client, id int) (*NSGroup, error) {
	path := fmt.Sprintf("/v1beta/dns/nameservers/groups/%d", id)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result GetNSGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
