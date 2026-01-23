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

func TestListNSGroups(t *testing.T) {
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

	groups, err := nsgroups.List(apiClient)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if groups == nil {
		t.Log("Note: No NS groups returned by mock server")
	}
}

func TestGetNSGroup(t *testing.T) {
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

	group, err := nsgroups.Get(apiClient, 1)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server")
	}
}
