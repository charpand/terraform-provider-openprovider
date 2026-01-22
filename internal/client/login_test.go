// Package authentication_test contains tests for the authentication package.
package client

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestLogin(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := Config{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
	client := NewClient(config)

	token, err := Login(client.HTTPClient, client.BaseURL, "127.0.0.1", "test", "test")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == nil {
		t.Log("Note: No token returned by mock server (check your swagger examples)")
	}
}
