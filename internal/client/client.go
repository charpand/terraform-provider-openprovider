// Package client provides a client for interacting with the OpenProvider API.
package client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charpand/terraform-provider-openprovider/internal/client/authentication"
)

const (
	// DefaultBaseURL -- root url for openprovider api
	DefaultBaseURL = "https://api.openprovider.eu"
)

// Config represents the configuration settings for a client, including the base API URL and an optional HTTP client.
type Config struct {
	BaseURL  string
	Username string
	Password string
	Token    string

	HTTPClient *http.Client
}

// Client represents a client for interacting with the OpenProvider API.
type Client struct {
	BaseURL  string
	Username string
	Password string
	Token    string

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
		// A registration can take longer than a request timeout sized for a
		// read: the first live one answered after the 30s the client used to
		// allow, so the order went through while the apply reported a failure
		// and recorded nothing.
		httpClient = &http.Client{
			Timeout: time.Second * 180,
		}
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
		Username:   config.Username,
		Password:   config.Password,
		Token:      config.Token,
	}
}

// Do executes a request and returns the response. It handles authentication and retries once if the token is expired.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.Token == "" && c.Username != "" && c.Password != "" {
		token, err := authentication.Login(c.HTTPClient, c.BaseURL, "", c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("initial authentication failed: %w", err)
		}
		c.Token = *token
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusUnauthorized && c.Username != "" && c.Password != "" {
		// Try to login and retry the request
		token, err := authentication.Login(c.HTTPClient, c.BaseURL, "", c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		c.Token = *token

		// Update Authorization header and retry
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The body carries the API's reason (`{"desc":...,"code":...}`);
		// without it a refusal reads as a bare status.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return resp, fmt.Errorf("api error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return resp, nil
}
