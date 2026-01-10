// Package authentication provides functionality for user authentication with the OpenProvider API.
package authentication

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/openprovider-go"
)

type mockTransport struct {
	rt http.RoundTripper
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer dummy")
	return t.rt.RoundTrip(req)
}

func TestLogin(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &mockTransport{rt: http.DefaultTransport},
	}

	config := openprovider.Config{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
	client := openprovider.NewClient(config)

	token, err := Login(client, "127.0.0.1", "test", "test")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == nil {
		t.Log("Note: No token returned by mock server (check your swagger examples)")
	}
}
