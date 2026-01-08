// Package openprovider provides a client for interacting with the OpenProvider API.
package openprovider

import (
	"net/http"
	"time"
)

const (
	// DefaultBaseURL -- root url for openprovider api
	DefaultBaseURL = "https://api.openprovider.eu"
)

// Config represents the configuration settings for a client, including the base API URL and an optional HTTP client.
type Config struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Client represents a client for interacting with the OpenProvider API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new client with the given configuration.
func NewClient(config Config) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Second * 30,
		}
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
}
