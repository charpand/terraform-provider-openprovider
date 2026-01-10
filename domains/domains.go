// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/charpand/openprovider-go"
)

// Domain represents a domain entity.
type Domain struct {
	ID     int `json:"id"`
	Domain struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
}

// ListDomainsResponse represents a response from the domains listing endpoint.
type ListDomainsResponse struct {
	Code int `json:"code"`
	Data struct {
		Results []Domain `json:"results"`
		Total   int      `json:"total"`
	} `json:"data"`
}

// List retrieves a list of domains from the Openprovider API.
func List(c *openprovider.Client) ([]Domain, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/domains", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
	}

	var results ListDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results.Data.Results, nil
}
