// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
)

// TestGetReportsAMissingGroup reproduces the #110-class regression:
// client.Client.Do now turns a 404 into a non-nil error, so a caller that
// only special-cased "an empty body means gone" (never reached, since Get
// propagated the error before decoding anything) would see a group deleted
// outside Terraform as a hard error instead of a clean removal from state.
func TestGetReportsAMissingGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	group, err := nsgroups.Get(c, "gone-group")
	if err != nil {
		t.Fatalf("expected a 404 to be reported as (nil, nil), not an error, got %v", err)
	}
	if group != nil {
		t.Fatalf("expected a 404 to report no group, got %+v", group)
	}
}

func TestGetReportsAFailedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	if _, err := nsgroups.Get(c, "some-group"); err == nil {
		t.Fatal("expected a non-404 failed request to still be reported as an error")
	}
}
