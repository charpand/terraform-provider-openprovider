package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockAuthTransport struct {
	loginCalled       bool
	return400OnNoAuth bool
}

func (m *mockAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Path, "/v1/auth/login") {
		m.loginCalled = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"code": 0, "data": {"token": "new-token"}}`)),
			Header:     make(http.Header),
		}, nil
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		status := http.StatusUnauthorized
		body := "Unauthorized"
		if m.return400OnNoAuth {
			status = http.StatusBadRequest
			body = "Bad Request"
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	if auth == "Bearer expired-token" {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`Unauthorized`)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
		Header:     make(http.Header),
	}, nil
}

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

	t.Run("Automatic login when token is missing", func(t *testing.T) {
		transport := &mockAuthTransport{return400OnNoAuth: true}
		hc := &http.Client{Transport: transport}
		client := NewClient(Config{
			Username:   "testuser",
			Password:   "testpass",
			HTTPClient: hc,
		})

		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, err := client.Do(req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %d", resp.StatusCode)
		}
		if !transport.loginCalled {
			t.Error("Expected Login to be called, but it wasn't")
		}
		if client.Token != "new-token" {
			t.Errorf("Expected token to be updated to 'new-token', got '%s'", client.Token)
		}
	})

	t.Run("Token expiration and retry", func(t *testing.T) {
		transport := &mockAuthTransport{}
		hc := &http.Client{Transport: transport}
		client := NewClient(Config{
			Username:   "testuser",
			Password:   "testpass",
			Token:      "expired-token",
			HTTPClient: hc,
		})

		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, err := client.Do(req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %d", resp.StatusCode)
		}
		if !transport.loginCalled {
			t.Error("Expected Login to be called on 401, but it wasn't")
		}
		if client.Token != "new-token" {
			t.Errorf("Expected token to be updated to 'new-token', got '%s'", client.Token)
		}
	})
}

// flakyTransport answers the first `failures` calls with `status` and every
// later one with 200, and records each body it was sent.
type flakyTransport struct {
	failures int
	status   int
	calls    int
	bodies   []string
}

func (f *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	f.bodies = append(f.bodies, body)
	status := http.StatusOK
	if f.calls <= f.failures {
		status = f.status
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Header:     make(http.Header),
	}, nil
}

func TestDoRepeatsAGatewayError(t *testing.T) {
	old := retryBase
	retryBase = 0
	t.Cleanup(func() { retryBase = old })

	t.Run("a GET answered by a 502 is sent again", func(t *testing.T) {
		transport := &flakyTransport{failures: 2, status: http.StatusBadGateway}
		client := NewClient(Config{Token: "token", HTTPClient: &http.Client{Transport: transport}})
		req, _ := http.NewRequest("GET", client.BaseURL+"/v1/domains", nil)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Expected the third call to answer, got %v", err)
		}
		_ = resp.Body.Close()
		if transport.calls != 3 {
			t.Errorf("Expected 3 calls, got %d", transport.calls)
		}
	})

	t.Run("a PUT is sent again with its body", func(t *testing.T) {
		transport := &flakyTransport{failures: 1, status: http.StatusServiceUnavailable}
		client := NewClient(Config{Token: "token", HTTPClient: &http.Client{Transport: transport}})
		req, _ := http.NewRequest("PUT", client.BaseURL+"/v1/domains/1", strings.NewReader(`{"a":1}`))

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Expected the second call to answer, got %v", err)
		}
		_ = resp.Body.Close()
		if len(transport.bodies) != 2 || transport.bodies[0] != `{"a":1}` || transport.bodies[1] != `{"a":1}` {
			t.Errorf("Expected the body on both calls, got %q", transport.bodies)
		}
	})

	t.Run("a POST answered by a 502 is not sent again", func(t *testing.T) {
		transport := &flakyTransport{failures: 1, status: http.StatusBadGateway}
		client := NewClient(Config{Token: "token", HTTPClient: &http.Client{Transport: transport}})
		req, _ := http.NewRequest("POST", client.BaseURL+"/v1/domains", strings.NewReader(`{}`))

		if _, err := client.Do(req); err == nil {
			t.Fatal("Expected the 502 to be an error")
		}
		if transport.calls != 1 {
			t.Errorf("Expected 1 call, got %d", transport.calls)
		}
	})

	t.Run("a GET that keeps failing is given up after the attempts", func(t *testing.T) {
		transport := &flakyTransport{failures: 10, status: http.StatusGatewayTimeout}
		client := NewClient(Config{Token: "token", HTTPClient: &http.Client{Transport: transport}})
		req, _ := http.NewRequest("GET", client.BaseURL+"/v1/domains", nil)

		if _, err := client.Do(req); err == nil {
			t.Fatal("Expected the last 504 to be an error")
		}
		if transport.calls != retryAttempts {
			t.Errorf("Expected %d calls, got %d", retryAttempts, transport.calls)
		}
	})

	t.Run("a 500 is not sent again", func(t *testing.T) {
		transport := &flakyTransport{failures: 1, status: http.StatusInternalServerError}
		client := NewClient(Config{Token: "token", HTTPClient: &http.Client{Transport: transport}})
		req, _ := http.NewRequest("GET", client.BaseURL+"/v1/domains", nil)

		if _, err := client.Do(req); err == nil {
			t.Fatal("Expected the 500 to be an error")
		}
		if transport.calls != 1 {
			t.Errorf("Expected 1 call, got %d", transport.calls)
		}
	})
}

func TestRetryDelay(t *testing.T) {
	old := retryBase
	retryBase = time.Second
	t.Cleanup(func() { retryBase = old })

	// Truncated to the second, so formatting it as an HTTP-date and parsing
	// it back loses nothing the comparisons below would notice.
	now := time.Now().Truncate(time.Second)

	if got := retryDelay(3, nil, now); got != 4*time.Second {
		t.Errorf("Expected the third pause to be 4s, got %v", got)
	}

	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Retry-After", "7")
	if got := retryDelay(1, resp, now); got != 7*time.Second {
		t.Errorf("Expected the header's 7s, got %v", got)
	}

	resp.Header.Set("Retry-After", "600")
	if got := retryDelay(1, resp, now); got != retryCap {
		t.Errorf("Expected the cap %v, got %v", retryCap, got)
	}

	resp.Header.Set("Retry-After", now.Add(5*time.Second).UTC().Format(http.TimeFormat))
	if got := retryDelay(1, resp, now); got != 5*time.Second {
		t.Errorf("Expected the header's date 5s ahead, got %v", got)
	}

	resp.Header.Set("Retry-After", now.Add(-5*time.Second).UTC().Format(http.TimeFormat))
	if got := retryDelay(1, resp, now); got != 0 {
		t.Errorf("Expected a date already past to mean no extra wait, got %v", got)
	}
}
