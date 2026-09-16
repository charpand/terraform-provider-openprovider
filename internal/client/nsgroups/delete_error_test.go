// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
)

func TestDeleteNSGroupReportsAFailedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":61,"desc":"Authentication failure"}`))
	}))
	defer server.Close()

	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
	if err := nsgroups.Delete(c, "test-group"); err == nil {
		t.Fatal("expected a failed delete request to be reported as an error")
	}
}
