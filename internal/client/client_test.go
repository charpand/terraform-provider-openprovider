package client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		client := NewClient(Config{})

		if client.BaseURL != DefaultBaseURL {
			t.Errorf("Expected BaseURL %s, got %s", DefaultBaseURL, client.BaseURL)
		}
		if client.HTTPClient == nil {
			t.Error("Expected default HTTPClient to be initialized, got nil")
		}
	})

	t.Run("Custom BaseURL", func(t *testing.T) {
		customURL := "http://localhost:4010"
		client := NewClient(Config{
			BaseURL: customURL,
		})

		if client.BaseURL != customURL {
			t.Errorf("Expected BaseURL %s, got %s", customURL, client.BaseURL)
		}
	})
}
