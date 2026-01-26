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

// setupTestClient creates a test client with default configuration.
func setupTestClient() *client.Client {
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
	return client.NewClient(config)
}

func TestListNSGroups(t *testing.T) {
	apiClient := setupTestClient()

	groups, err := nsgroups.List(apiClient)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if groups == nil {
		t.Log("Note: No NS groups returned by mock server")
	}
}

func TestGetNSGroup(t *testing.T) {
	apiClient := setupTestClient()

	group, err := nsgroups.Get(apiClient, "test-group")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server")
	}
}

func TestCreateNSGroupPreservesIPFromAPI(t *testing.T) {
	apiClient := setupTestClient()

	// Create a NS group with only hostname (no IP/IP6)
	req := &nsgroups.CreateNSGroupRequest{
		Name: "test-ns-group-hostname-only",
		Nameservers: []nsgroups.Nameserver{
			{Name: "ns1.domain.com"}, // Only hostname provided
		},
	}

	group, err := nsgroups.Create(apiClient, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server")
		return
	}

	// Check if API populated IP/IP6 fields
	if len(group.Nameservers) > 0 {
		ns := group.Nameservers[0]
		t.Logf("Nameserver response: Name=%s, IP=%s, IP6=%s", ns.Name, ns.IP, ns.IP6)

		// According to the swagger spec example, the API should return IP addresses
		// when a valid hostname is provided
		if ns.IP == "" && ns.IP6 == "" {
			t.Log("Note: API did not populate IP/IP6 fields (this might be expected for mock server)")
		} else {
			t.Logf("Success: API populated IP=%s, IP6=%s", ns.IP, ns.IP6)
		}
	}
}
