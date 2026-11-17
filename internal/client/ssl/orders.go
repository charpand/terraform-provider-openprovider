// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
		httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
		if err != nil {
			return nil, err
		}

		page, total, err := listOrdersPage(c, httpReq)
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

func listOrdersPage(c *client.Client, httpReq *http.Request) ([]SSLOrder, int, error) {
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, 0, err
	}

	var result ListSSLOrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	return result.Data.Results, result.Data.Total, nil
}

// GetOrder retrieves a specific SSL order by ID.
//
// Endpoint: GET https://api.openprovider.eu/v1/ssl/orders/{id}
func GetOrder(c *client.Client, orderID int) (*SSLOrder, error) {
	path := fmt.Sprintf("/v1/ssl/orders/%d", orderID)
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

	var result GetSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateOrder creates a new SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders
func CreateOrder(c *client.Client, req *CreateSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/v1/ssl/orders"
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result CreateSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateOrder updates an existing SSL order (e.g., autorenew settings).
//
// Endpoint: PATCH https://api.openprovider.eu/v1/ssl/orders/{id}
func UpdateOrder(c *client.Client, orderID int, req *UpdateSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/ssl/orders/%d", orderID)
	httpReq, err := http.NewRequest("PATCH", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result UpdateSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// RenewOrder renews an existing SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders/{id}/renew
func RenewOrder(c *client.Client, orderID int, req *RenewSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/ssl/orders/%d/renew", orderID)
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result RenewSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// ReissueOrder reissues an existing SSL order.
//
// Endpoint: POST https://api.openprovider.eu/v1/ssl/orders/{id}/reissue
func ReissueOrder(c *client.Client, orderID int, req *ReissueSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/ssl/orders/%d/reissue", orderID)
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result ReissueSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CancelOrder cancels an SSL order.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/ssl/orders/{id}
func CancelOrder(c *client.Client, orderID int) error {
	path := fmt.Sprintf("/v1/ssl/orders/%d", orderID)
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return err
	}

	var result CancelSSLOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	return nil
}
