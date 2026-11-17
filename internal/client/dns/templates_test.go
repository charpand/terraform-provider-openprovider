package dns

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

func templateClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})
}

func TestCreateTemplateSendsTheRequest(t *testing.T) {
	var got map[string]any
	c := templateClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/dns/templates" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"id":42}}`)
	})

	id, err := CreateTemplate(c, &CreateTemplateRequest{
		Name:    "web",
		Records: []TemplateRecord{{Name: "www", Type: "A", Value: "192.0.2.1", TTL: 3600}, {Name: "", Type: "MX", Value: "mail.example.com", TTL: 300, Priority: 10}},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if id != 42 {
		t.Errorf("id: got %d, want 42", id)
	}
	records, _ := got["records"].([]any)
	if got["name"] != "web" || len(records) != 2 {
		t.Errorf("body: got %v", got)
	}
	if mx, _ := records[1].(map[string]any); mx["prio"] != float64(10) {
		t.Errorf("mx record: got %v", records[1])
	}
}

func TestCreateTemplateRefused(t *testing.T) {
	for name, answer := range map[string]string{
		"code in a 2xx": `{"code":399,"desc":"refused"}`,
		"no id":         `{"code":0,"data":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := templateClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, answer) })
			if _, err := CreateTemplate(c, &CreateTemplateRequest{Name: "web"}); err == nil {
				t.Error("CreateTemplate should fail")
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	c := templateClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/dns/templates/42" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"id":42,"name":"web","records":[{"id":1,"name":"www","type":"A","value":"192.0.2.1","ttl":3600}]}}`)
	})
	tmpl, err := GetTemplate(c, 42)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if tmpl.ID != 42 || tmpl.Name != "web" || len(tmpl.Records) != 1 || tmpl.Records[0].Value != "192.0.2.1" {
		t.Errorf("template: got %+v", tmpl)
	}
}

func TestDeleteTemplate(t *testing.T) {
	c := templateClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/dns/templates/42" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":true}}`)
	})
	if err := DeleteTemplate(c, 42); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	refused := templateClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"success":false}}`)
	})
	if err := DeleteTemplate(refused, 42); err == nil {
		t.Error("DeleteTemplate should fail without success")
	}
}
