// Package domains_test contains tests for the domains package.
package domains_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestGetAuthCode(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	code, err := domains.GetAuthCode(apiClient, 123, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if code == nil || code.AuthCode == "" {
		t.Fatalf("Expected an auth code from the mock server, got %v", code)
	}
}

func TestResetAuthCode(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	code, err := domains.ResetAuthCode(apiClient, 123, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if code == nil || code.AuthCode == "" {
		t.Fatalf("Expected an auth code from the mock server, got %v", code)
	}
}

// TestAuthCodeRequests checks the method, the path, the query and the body
// that each function sends, for each auth code type.
func TestAuthCodeRequests(t *testing.T) {
	type request struct {
		method, path, query string
		body                map[string]any
	}
	var got request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = request{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery}
		if r.Method == http.MethodPost {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			if err := json.Unmarshal(raw, &got.body); err != nil {
				t.Errorf("decode body %s: %v", raw, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"auth_code":"secret","type":"external"}}`)
	}))
	defer srv.Close()
	c := client.NewClient(client.Config{BaseURL: srv.URL, Token: "test"})

	for _, tc := range []struct {
		name string
		call func() (*domains.AuthCode, error)
		want request
	}{
		{
			name: "get",
			call: func() (*domains.AuthCode, error) { return domains.GetAuthCode(c, 7, "") },
			want: request{method: "GET", path: "/v1beta/domains/7/authcode"},
		},
		{
			name: "get internal",
			call: func() (*domains.AuthCode, error) {
				return domains.GetAuthCode(c, 7, domains.AuthCodeTypeInternal)
			},
			want: request{method: "GET", path: "/v1beta/domains/7/authcode", query: "auth_code_type=internal"},
		},
		{
			name: "reset",
			call: func() (*domains.AuthCode, error) { return domains.ResetAuthCode(c, 7, "") },
			want: request{method: "POST", path: "/v1beta/domains/7/authcode/reset", body: map[string]any{"id": float64(7)}},
		},
		{
			name: "reset internal",
			call: func() (*domains.AuthCode, error) {
				return domains.ResetAuthCode(c, 7, domains.AuthCodeTypeInternal)
			},
			want: request{
				method: "POST",
				path:   "/v1beta/domains/7/authcode/reset",
				body:   map[string]any{"id": float64(7), "auth_code_type": "internal"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got = request{}
			code, err := tc.call()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if code.AuthCode != "secret" {
				t.Errorf("auth code: got %q, want %q", code.AuthCode, "secret")
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("request: got %+v, want %+v", got, tc.want)
			}
		})
	}
}
