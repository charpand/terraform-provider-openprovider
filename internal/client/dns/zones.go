// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"encoding/json"
	"fmt"
	"net/http"

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
		httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
		if err != nil {
			return nil, err
		}

		page, total, err := doList[ListZonesResponse, Zone](c, httpReq, func(r ListZonesResponse) ([]Zone, int) {
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
func doList[R any, T any](c *client.Client, httpReq *http.Request, unpack func(R) ([]T, int)) ([]T, int, error) {
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, 0, err
	}

	var result R
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}
	page, total := unpack(result)
	return page, total, nil
}

// GetZone retrieves a specific DNS zone by name.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/zones/{name}
func GetZone(c *client.Client, zoneName string) (*Zone, error) {
	path := fmt.Sprintf("/v1/dns/zones/%s", zoneName)
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
