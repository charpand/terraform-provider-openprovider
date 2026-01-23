// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// GetNSGroupRequest represents a request to retrieve a nameserver group.
type GetNSGroupRequest struct {
	ID int `json:"id"`
}

// GetNSGroupByNameResponse represents a response for getting NS groups by name.
type GetNSGroupByNameResponse struct {
	Code int `json:"code"`
	Data struct {
		Results []NSGroup `json:"results"`
		Total   int       `json:"total"`
	} `json:"data"`
}

// GetByName retrieves a nameserver group by name from the Openprovider API.
// This is useful for import operations where the name is known but not the ID.
func GetByName(c *client.Client, name string) (*NSGroup, error) {
	path := fmt.Sprintf("/v1beta/dns/nameservers/groups?name_pattern=%s", name)
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

	var results GetNSGroupByNameResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	// Find exact match
	for _, group := range results.Data.Results {
		if group.Name == name {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("nameserver group with name '%s' not found", name)
}
