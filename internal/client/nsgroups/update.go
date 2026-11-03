// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// UpdateNSGroupRequest represents a request to update a nameserver group.
type UpdateNSGroupRequest struct {
	Name        string       `json:"ns_group,omitempty"`
	Nameservers []Nameserver `json:"name_servers,omitempty"`
}

// UpdateNSGroupResponse represents a response for updating a nameserver group.
type UpdateNSGroupResponse struct {
	Code int     `json:"code"`
	Data NSGroup `json:"data"`
}

// Update updates an existing nameserver group by name via the Openprovider API.
//
// Endpoint: PUT https://api.openprovider.eu/v1/dns/nameservers/groups/{ns_group}
func Update(c *client.Client, name string, req *UpdateNSGroupRequest) (*NSGroup, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/dns/nameservers/groups/%s", name)
	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
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

	var result UpdateNSGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
