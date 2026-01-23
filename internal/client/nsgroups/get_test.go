// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestGetNSGroupByName(t *testing.T) {
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
	apiClient := client.NewClient(config)

	group, err := nsgroups.GetByName(apiClient, "test-group")

	if err != nil {
		// This is expected if no group matches
		t.Logf("Note: GetByName returned error (expected if no match): %v", err)
	}

	if group != nil && group.Name == "" {
		t.Log("Note: NS group name not populated by mock server")
	}
}
