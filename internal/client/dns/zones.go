// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListZones lists all DNS zones.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/dns/zones
func ListZones(c *client.Client) ([]Zone, error) {
	path := "/v1beta/dns/zones"
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result ListZonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Results, nil
}

// GetZone retrieves a specific DNS zone by name.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/dns/zones/{name}
func GetZone(c *client.Client, zoneName string) (*Zone, error) {
	path := fmt.Sprintf("/v1beta/dns/zones/%s", zoneName)
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result GetZoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
