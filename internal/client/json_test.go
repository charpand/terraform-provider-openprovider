package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonTestClient(status int, answer string, seen *http.Request, seenBody *string) *Client {
	return &Client{
		BaseURL: "http://api.test",
		Token:   "t",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if seen != nil {
				*seen = *r
			}
			if r.Body != nil && seenBody != nil {
				b, _ := io.ReadAll(r.Body)
				*seenBody = string(b)
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(answer)),
				Header:     make(http.Header),
			}, nil
		})},
	}
}

func TestJSONSendsBodyAndDecodesAnswer(t *testing.T) {
	var req http.Request
	var body string
	c := jsonTestClient(200, `{"data":{"id":7}}`, &req, &body)

	var out struct{ Data struct{ ID int } }
	if err := c.JSON("POST", "/v1/things", map[string]string{"a": "b"}, &out); err != nil {
		t.Fatal(err)
	}

	if req.Method != http.MethodPost || req.URL.String() != "http://api.test/v1/things" {
		t.Errorf("sent %s %s", req.Method, req.URL)
	}
	if body != `{"a":"b"}` {
		t.Errorf("body = %q", body)
	}
	if out.Data.ID != 7 {
		t.Errorf("decoded id = %d, want 7", out.Data.ID)
	}
}

func TestJSONWithoutBodyOrOut(t *testing.T) {
	var body string
	c := jsonTestClient(200, `not json`, nil, &body)
	if err := c.JSON("DELETE", "/v1/things/1", nil, nil); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		t.Errorf("a nil body sent %q", body)
	}
}

func TestJSONDecodeError(t *testing.T) {
	c := jsonTestClient(200, `not json`, nil, nil)
	var out map[string]any
	if err := c.JSON("GET", "/x", nil, &out); err == nil {
		t.Fatal("want a decode error")
	}
}

func TestJSONEncodeError(t *testing.T) {
	c := jsonTestClient(200, `{}`, nil, nil)
	if err := c.JSON("POST", "/x", func() {}, nil); err == nil {
		t.Fatal("want an encode error")
	}
}

func TestAPIErrorAndIsNotFound(t *testing.T) {
	c := jsonTestClient(404, ` {"desc":"gone","code":399} `, nil, nil)
	err := c.JSON("GET", "/x", nil, nil)

	if !IsNotFound(err) {
		t.Fatalf("IsNotFound(%v) = false", err)
	}
	if !IsNotFound(fmt.Errorf("wrapped: %w", err)) {
		t.Error("IsNotFound must see through wrapping")
	}
	want := `api error: status 404: {"desc":"gone","code":399}`
	if err.Error() != want {
		t.Errorf("message = %q, want %q", err.Error(), want)
	}

	c = jsonTestClient(500, `boom`, nil, nil)
	if IsNotFound(c.JSON("GET", "/x", nil, nil)) {
		t.Error("a 500 is not a not-found")
	}
	if IsNotFound(nil) || IsNotFound(errors.New("other")) {
		t.Error("only an APIError can be a not-found")
	}
}
