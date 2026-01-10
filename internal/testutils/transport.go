// Package testutils provides helpers for testing the openprovider client.
package testutils

import "net/http"

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
