// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListProducts lists all available SSL products.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/ssl/products
func ListProducts(c *client.Client) ([]SSLProduct, error) {
	path := "/v1beta/ssl/products"
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

	var result ListSSLProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Results, nil
}

// GetProduct retrieves a specific SSL product by ID.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/ssl/products/{id}
func GetProduct(c *client.Client, productID int) (*SSLProduct, error) {
	path := fmt.Sprintf("/v1beta/ssl/products/%d", productID)
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

	var result GetSSLProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
