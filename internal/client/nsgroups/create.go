// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateNSGroupRequest represents a request to create a nameserver group.
type CreateNSGroupRequest struct {
	Name        string       `json:"name"`
	Nameservers []Nameserver `json:"name_servers"`
}

// CreateNSGroupResponse represents a response for creating a nameserver group.
type CreateNSGroupResponse struct {
	Code int     `json:"code"`
	Data NSGroup `json:"data"`
}

// Create creates a new nameserver group via the Openprovider API.
//
// Endpoint: POST https://api.eu/v1beta/dns/nameservers/groups
func Create(c *client.Client, req *CreateNSGroupRequest) (*NSGroup, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/dns/nameservers/groups"
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

	var result CreateNSGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
