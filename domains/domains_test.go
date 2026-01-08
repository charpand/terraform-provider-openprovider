package domains

import (
	"os"
	"testing"

	"github.com/charpand/openprovider-go"
)

func TestListDomains(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	config := openprovider.Config{
		BaseURL: baseURL,
	}
	client := openprovider.NewClient(config)

	domains, err := List(client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(domains) == 0 {
		t.Log("Note: No domains returned by mock server (check your swagger examples)")
	}
}
