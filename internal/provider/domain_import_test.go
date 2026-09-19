package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// domainStatusAPI stands in for OpenProvider with example.com in the account
// under the status given.
func domainStatusAPI(t *testing.T, status string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/domains" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"results": []map[string]any{{
					"id":     123,
					"domain": map[string]string{"name": "example", "extension": "com"},
					"status": status,
				}},
				"total": 1,
			},
		})
	}))
}

// importDomain imports example.com from an account which holds it under
// `status`, and reports the summary of every warning the import raised.
func importDomain(t *testing.T, status string) (warnings []string, errored bool) {
	t.Helper()
	ctx := context.Background()
	server := domainStatusAPI(t, status)
	defer server.Close()
	r := &DomainResource{client: client.NewClient(client.Config{
		BaseURL:    server.URL,
		Token:      "test",
		HTTPClient: server.Client(),
	})}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	// The null object the framework hands an import before the resource fills
	// its attributes in.
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "example.com"}, resp)
	for _, d := range resp.Diagnostics.Warnings() {
		warnings = append(warnings, d.Summary())
	}
	return warnings, resp.Diagnostics.HasError()
}

// A name the account holds outright needs no authorization code, so the import
// of one says nothing. The operator lane imports the domains it already owns on
// every deploy, and a warning there is noise on an apply that is in order.
func TestImportOfAnOwnedDomainIsQuiet(t *testing.T) {
	warnings, errored := importDomain(t, "ACT")
	if errored {
		t.Fatal("import of an owned domain must not fail")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

// A transfer which is not complete is the case the warning is for: the code
// that started it cannot be read back from the API.
func TestImportOfADomainInTransferAsksForTheAuthCode(t *testing.T) {
	warnings, errored := importDomain(t, "REQ")
	if errored {
		t.Fatal("import of a domain in transfer must not fail")
	}
	if len(warnings) != 1 || warnings[0] != "Auth Code Required for Transferred Domains" {
		t.Errorf("expected the auth-code warning, got %v", warnings)
	}
}
