package dns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

func TestGetRecordWithValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"results":[`+
			`{"name":"example.com","type":"MX","value":"mail1.example.com","ttl":300},`+
			`{"name":"example.com","type":"MX","value":"mail2.example.com","ttl":600}],"total":2}}`)
	}))
	t.Cleanup(srv.Close)
	c := client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})

	for name, tc := range map[string]struct {
		value   string
		wantTTL int
		wantErr bool
		rname   string
	}{
		"the second value": {"mail2.example.com", 600, false, "example.com"},
		"the first value":  {"mail1.example.com", 300, false, "example.com"},
		"an unknown value": {"gone.example.com", 300, false, "example.com"},
		"an unknown name":  {"mail1.example.com", 0, true, "other.example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			record, err := GetRecordWithValue(c, "example.com", tc.rname, "MX", tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatal("it should fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetRecordWithValue: %v", err)
			}
			if record.TTL != tc.wantTTL {
				t.Errorf("ttl: got %d, want %d", record.TTL, tc.wantTTL)
			}
		})
	}
}
