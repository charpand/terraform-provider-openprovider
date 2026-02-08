// Package testutils provides helpers for testing the openprovider client.
package testutils

import (
	"fmt"
	"net/http"
)

// MockTransport is a custom http.RoundTripper for testing against the Prism mock server.
type MockTransport struct {
	RT http.RoundTripper
}

// RoundTrip adds the necessary headers for Prism mock server validation.
func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer dummy")
	req.Header.Set("Prefer", "code=200")
	return t.RT.RoundTrip(req)
}

// ErrorMockTransport is a custom http.RoundTripper for testing error scenarios.
type ErrorMockTransport struct {
	RT         http.RoundTripper
	StatusCode int
}

// RoundTrip adds headers to request a specific error status code from Prism.
func (t *ErrorMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer dummy")
	req.Header.Set("Prefer", fmt.Sprintf("code=%d", t.StatusCode))
	return t.RT.RoundTrip(req)
}
