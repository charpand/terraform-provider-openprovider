// Package client provides a client for interacting with the OpenProvider API.
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charpand/terraform-provider-openprovider/internal/client/authentication"
)

const (
	// DefaultBaseURL -- root url for openprovider api
	DefaultBaseURL = "https://api.openprovider.eu"

	// A request the gateway answers with a 5xx, or does not answer at all,
	// is sent this many times in total before Do gives up on it (so it is
	// repeated retryAttempts-1 times after the first try).
	retryAttempts = 4
	// The longest pause a `Retry-After` header can ask for.
	retryCap = 30 * time.Second
)

// The pause before the first repeat; it doubles on each further one. A
// variable so the tests can shorten it.
var retryBase = time.Second

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

// Do executes a request and returns the response. It handles authentication
// and retries once if the token is expired. A request the gateway answers
// with a 429, 502, 503 or 504, or does not answer at all, is repeated after
// a pause when its method allows: a `POST` that got a gateway error may have
// been handled, and a repeat could order twice.
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

	var resp *http.Response
	var err error
	for attempt := 1; ; attempt++ {
		resp, err = c.send(req)
		if err == nil && resp.StatusCode == http.StatusUnauthorized && c.Username != "" && c.Password != "" {
			_ = resp.Body.Close()
			// Try to login and retry the request
			token, loginErr := authentication.Login(c.HTTPClient, c.BaseURL, "", c.Username, c.Password)
			if loginErr != nil {
				return nil, fmt.Errorf("authentication failed: %w", loginErr)
			}
			c.Token = *token

			// Update Authorization header and retry
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
			resp, err = c.send(req)
		}

		if attempt == retryAttempts || !transient(resp, err) || !repeatable(req) {
			break
		}
		delay := retryDelay(attempt, resp, time.Now())
		if resp != nil {
			_ = resp.Body.Close()
		}
		if waitErr := wait(req.Context(), delay); waitErr != nil {
			return nil, waitErr
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

// send sends the request once. A body the request carries is rewound
// first, so a repeat sends it again instead of the consumed reader.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		req.Body = body
	}
	return c.HTTPClient.Do(req)
}

// transient reports whether the answer is one the API did not give: a
// transport error, a rate limit, or a gateway error.
func transient(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// repeatable reports whether the request can be sent again without a
// second effect: an idempotent method, and a body that can be rewound.
func repeatable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	return req.Body == nil || req.GetBody != nil
}

// retryDelay is the pause before the repeat after the given attempt: what a
// `Retry-After` header asks for, up to `retryCap`, else `retryBase` doubled
// per attempt. RFC 7231 allows that header as either a number of seconds or
// an HTTP-date; `now` is what the date is measured against, passed in
// rather than read from the clock so this stays deterministic to test.
func retryDelay(attempt int, resp *http.Response, now time.Time) time.Duration {
	if resp != nil {
		if raw := resp.Header.Get("Retry-After"); raw != "" {
			if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
				// Compare in whole seconds first: converting a huge value
				// to a Duration overflows to a negative pause.
				if seconds >= int(retryCap/time.Second) {
					return retryCap
				}
				return time.Duration(seconds) * time.Second
			}
			if when, err := http.ParseTime(raw); err == nil {
				if until := when.Sub(now); until > 0 {
					return min(until, retryCap)
				}
				// A date already past asks for no extra wait, not the
				// exponential fallback below.
				return 0
			}
		}
	}
	return retryBase << (attempt - 1)
}

// wait pauses for the delay, or until the request's context ends.
func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
