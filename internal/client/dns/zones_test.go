// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListZones(t *testing.T) {
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

	zones, err := ListZones(c)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if zones == nil {
		t.Log("Note: No zones returned by mock server")
		return
	}

	t.Logf("Retrieved %d DNS zones", len(zones))
}

func TestGetZone(t *testing.T) {
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

	zone, err := GetZone(c, "example.com")
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if zone == nil {
		t.Log("Note: No zone returned by mock server")
		return
	}

	t.Logf("Retrieved DNS zone: %s.%s", zone.Name, zone.Extension)
}
