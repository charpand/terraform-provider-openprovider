// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// listPageSize is how many orders one request asks for while ListOrders pages
// through the account. The API's default page is 100, so listing everything
// must ask more than once; this is the size of each of those requests, not a
// cap on the result. A var so the tests can shrink it.
var listPageSize = 100

// ListOrders lists all SSL orders of the account, paging through the listing.
//
// Endpoint: GET https://api.openprovider.eu/v1/ssl/orders
func ListOrders(c *client.Client) ([]SSLOrder, error) {
	all := make([]SSLOrder, 0)
	for offset := 0; ; {
		path := fmt.Sprintf("/v1/ssl/orders?limit=%d&offset=%d", listPageSize, offset)
		page, total, err := listOrdersPage(c, path)
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

func listOrdersPage(c *client.Client, path string) ([]SSLOrder, int, error) {
	var result ListSSLOrdersResponse
	if err := c.JSON("GET", path, nil, &result); err != nil {
		return nil, 0, err
	}

	return result.Data.Results, result.Data.Total, nil
}

// GetOrder retrieves a specific SSL order by ID.
//
// Endpoint: GET https://api.openprovider.eu/v1/ssl/orders/{id}
func GetOrder(c *client.Client, orderID int) (*SSLOrder, error) {
	var result GetSSLOrderResponse
	if err := c.JSON("GET", fmt.Sprintf("/v1/ssl/orders/%d", orderID), nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateOrder creates a new SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders
func CreateOrder(c *client.Client, req *CreateSSLOrderRequest) (*SSLOrder, error) {
	var result CreateSSLOrderResponse
	if err := c.JSON("POST", "/v1/ssl/orders", req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateOrder updates an existing SSL order (e.g., autorenew settings).
//
// Endpoint: PATCH https://api.openprovider.eu/v1/ssl/orders/{id}
func UpdateOrder(c *client.Client, orderID int, req *UpdateSSLOrderRequest) (*SSLOrder, error) {
	var result UpdateSSLOrderResponse
	if err := c.JSON("PATCH", fmt.Sprintf("/v1/ssl/orders/%d", orderID), req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// RenewOrder renews an existing SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders/{id}/renew
func RenewOrder(c *client.Client, orderID int, req *RenewSSLOrderRequest) (*SSLOrder, error) {
	var result RenewSSLOrderResponse
	if err := c.JSON("POST", fmt.Sprintf("/v1/ssl/orders/%d/renew", orderID), req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// ReissueOrder reissues an existing SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders/{id}/reissue
func ReissueOrder(c *client.Client, orderID int, req *ReissueSSLOrderRequest) (*SSLOrder, error) {
	var result ReissueSSLOrderResponse
	if err := c.JSON("POST", fmt.Sprintf("/v1/ssl/orders/%d/reissue", orderID), req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CancelOrder cancels an SSL order.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/ssl/orders/{id}
func CancelOrder(c *client.Client, orderID int) error {
	var result CancelSSLOrderResponse
	return c.JSON("DELETE", fmt.Sprintf("/v1/ssl/orders/%d", orderID), nil, &result)
}
