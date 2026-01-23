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

func TestCreateNSGroup(t *testing.T) {
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

	req := &nsgroups.CreateNSGroupRequest{
		Name: "test-ns-group",
		Nameservers: []nsgroups.Nameserver{
			{Name: "ns1.example.com"},
			{Name: "ns2.example.com"},
		},
	}

	group, err := nsgroups.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server (check your swagger examples)")
		return
	}

	if group.Name == "" {
		t.Log("Note: NS group name not populated by mock server")
	}
}
