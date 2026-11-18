// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListProducts lists all available SSL products.
//
// Endpoint: GET https://api.openprovider.eu/v1/ssl/products
func ListProducts(c *client.Client) ([]SSLProduct, error) {
	var result ListSSLProductsResponse
	if err := c.JSON("GET", "/v1/ssl/products", nil, &result); err != nil {
		return nil, err
	}

	return result.Data.Results, nil
}

// GetProduct retrieves a specific SSL product by ID.
//
// Endpoint: GET https://api.openprovider.eu/v1/ssl/products/{id}
func GetProduct(c *client.Client, productID int) (*SSLProduct, error) {
	var result GetSSLProductResponse
	if err := c.JSON("GET", fmt.Sprintf("/v1/ssl/products/%d", productID), nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
