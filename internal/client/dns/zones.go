// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"fmt"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// listPageSize is how many results one request asks for while a list pages
// through the account. The API's default page is 100, so listing everything
// must ask more than once; this is the size of each of those requests, not a
// cap on the result. A var so the tests can shrink it.
var listPageSize = 100

// ListZones lists all DNS zones of the account, paging through the listing.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/zones
func ListZones(c *client.Client) ([]Zone, error) {
	all := make([]Zone, 0)
	for offset := 0; ; {
		path := fmt.Sprintf("/v1/dns/zones?limit=%d&offset=%d", listPageSize, offset)
		page, total, err := doList[ListZonesResponse, Zone](c, path, func(r ListZonesResponse) ([]Zone, int) {
			return r.Data.Results, r.Data.Total
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		offset += len(page)
		// A short or empty page stops the loop even if total disagrees, so a
		// bad or missing total can't spin this forever.
		if len(page) < listPageSize || offset >= total {
			return all, nil
		}
	}
}

// doList sends a listing request and decodes the page that it answers with.
func doList[R any, T any](c *client.Client, path string, unpack func(R) ([]T, int)) ([]T, int, error) {
	var result R
	if err := c.JSON("GET", path, nil, &result); err != nil {
		return nil, 0, err
	}
	page, total := unpack(result)
	return page, total, nil
}

// GetZone retrieves a specific DNS zone by name.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/zones/{name}
func GetZone(c *client.Client, zoneName string) (*Zone, error) {
	var result GetZoneResponse
	if err := c.JSON("GET", fmt.Sprintf("/v1/dns/zones/%s", zoneName), nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateZoneRequest represents a request to create a DNS zone.
type CreateZoneRequest struct {
	Domain       ZoneDomain `json:"domain"`
	Type         string     `json:"type,omitempty"`
	Provider     string     `json:"provider,omitempty"`
	Secured      *bool      `json:"secured,omitempty"`
	MasterIP     string     `json:"master_ip,omitempty"`
	TemplateName string     `json:"template_name,omitempty"`
	Records      []Record   `json:"records,omitempty"`
}

// ZoneDomain is the name and extension of the domain of a zone.
type ZoneDomain struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

type zoneResultResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Success *bool `json:"success"`
	} `json:"data"`
}

func (r zoneResultResponse) err(action string) error {
	if r.Code != 0 {
		return fmt.Errorf("%s failed with code %d: %s", action, r.Code, r.Desc)
	}
	if r.Data.Success != nil && !*r.Data.Success {
		return fmt.Errorf("%s was not successful: %s", action, r.Desc)
	}
	return nil
}

// CreateZone creates a DNS zone.
//
// Endpoint: POST https://api.openprovider.eu/v1/dns/zones
func CreateZone(c *client.Client, req *CreateZoneRequest) error {
	var result zoneResultResponse
	if err := c.JSON("POST", "/v1/dns/zones", req, &result); err != nil {
		return err
	}
	return result.err("create zone")
}

// DeleteZone deletes a DNS zone and all of its records.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/dns/zones/{name}
func DeleteZone(c *client.Client, zoneName string) error {
	var result zoneResultResponse
	if err := c.JSON("DELETE", fmt.Sprintf("/v1/dns/zones/%s", url.PathEscape(zoneName)), nil, &result); err != nil {
		return err
	}
	return result.err("delete zone")
}
