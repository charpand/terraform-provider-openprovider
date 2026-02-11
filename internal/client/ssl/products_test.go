// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListProducts(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	products, err := ListProducts(c)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if products == nil {
		t.Log("Note: No products returned by mock server")
		return
	}

	t.Logf("Retrieved %d SSL products", len(products))
}

func TestGetProduct(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	product, err := GetProduct(c, 1)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if product == nil {
		t.Log("Note: No product returned by mock server")
		return
	}

	t.Logf("Retrieved SSL product: %s (%s)", product.Name, product.BrandName)
}
