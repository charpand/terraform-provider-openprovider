package dns

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

func zoneWriteClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})
}

func TestCreateZoneSendsTheRequest(t *testing.T) {
	var got map[string]any
	c := zoneWriteClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/dns/zones" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
	})

	secured := true
	err := CreateZone(c, &CreateZoneRequest{
		Domain:       ZoneDomain{Name: "example", Extension: "com"},
		Type:         "master",
		Secured:      &secured,
		TemplateName: "default",
	})
	if err != nil {
		t.Fatalf("CreateZone: %v", err)
	}
	domain, _ := got["domain"].(map[string]any)
	if domain["name"] != "example" || domain["extension"] != "com" {
		t.Errorf("domain: got %v", got["domain"])
	}
	if got["type"] != "master" || got["secured"] != true || got["template_name"] != "default" {
		t.Errorf("body: got %v", got)
	}
	if _, ok := got["master_ip"]; ok {
		t.Errorf("an empty master_ip should not be sent: %v", got)
	}
}

func TestZoneWriteRefused(t *testing.T) {
	for name, answer := range map[string]string{
		"code in a 2xx": `{"code":399,"desc":"refused"}`,
		"no success":    `{"code":0,"data":{"success":false}}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := zoneWriteClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, answer) })
			if err := CreateZone(c, &CreateZoneRequest{}); err == nil {
				t.Error("CreateZone should fail")
			}
			if err := DeleteZone(c, "example.com"); err == nil {
				t.Error("DeleteZone should fail")
			}
		})
	}
}

func TestDeleteZoneSendsTheRequest(t *testing.T) {
	c := zoneWriteClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/dns/zones/example.com" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
	})
	if err := DeleteZone(c, "example.com"); err != nil {
		t.Fatalf("DeleteZone: %v", err)
	}
}
