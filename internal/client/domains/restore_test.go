package domains_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
)

func restoreClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})
}

func TestRestoreSendsTheRequest(t *testing.T) {
	var got map[string]any
	c := restoreClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/domains/7/restore" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"status":"PEN"}}`)
	})

	status, err := domains.Restore(c, 7, "example", "com")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if status != "PEN" {
		t.Errorf("status: got %q, want PEN", status)
	}
	domain, _ := got["domain"].(map[string]any)
	if got["id"] != float64(7) || domain["name"] != "example" || domain["extension"] != "com" {
		t.Errorf("body: got %v", got)
	}
}

func TestRestoreRefused(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		answer string
	}{
		"code in a 2xx": {http.StatusOK, `{"code":399,"desc":"not restorable"}`},
		"server error":  {http.StatusInternalServerError, `{}`},
	} {
		t.Run(name, func(t *testing.T) {
			c := restoreClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.answer)
			})
			if _, err := domains.Restore(c, 7, "example", "com"); err == nil {
				t.Error("Restore should fail")
			}
		})
	}
}
