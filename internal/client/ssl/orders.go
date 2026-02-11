// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListOrders lists all SSL orders.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/ssl/orders
func ListOrders(c *client.Client) ([]SSLOrder, error) {
	path := "/v1beta/ssl/orders"
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

	var result ListSSLOrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Results, nil
}

// GetOrder retrieves a specific SSL order by ID.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/ssl/orders/{id}
func GetOrder(c *client.Client, orderID int) (*SSLOrder, error) {
	path := fmt.Sprintf("/v1beta/ssl/orders/%d", orderID)
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
// Endpoint: POST https://api.openprovider.eu/v1beta/ssl/orders
func CreateOrder(c *client.Client, req *CreateSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/ssl/orders"
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
// Endpoint: PATCH https://api.openprovider.eu/v1beta/ssl/orders/{id}
func UpdateOrder(c *client.Client, orderID int, req *UpdateSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/ssl/orders/%d", orderID)
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
// Endpoint: POST https://api.openprovider.eu/v1beta/ssl/orders/{id}/renew
func RenewOrder(c *client.Client, orderID int, req *RenewSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/ssl/orders/%d/renew", orderID)
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
// Endpoint: POST https://api.openprovider.eu/v1beta/ssl/orders/{id}/reissue
func ReissueOrder(c *client.Client, orderID int, req *ReissueSSLOrderRequest) (*SSLOrder, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/ssl/orders/%d/reissue", orderID)
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
// Endpoint: DELETE https://api.openprovider.eu/v1beta/ssl/orders/{id}
func CancelOrder(c *client.Client, orderID int) error {
	path := fmt.Sprintf("/v1beta/ssl/orders/%d", orderID)
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
