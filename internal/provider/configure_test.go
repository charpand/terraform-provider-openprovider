package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

func TestClientFromProviderData(t *testing.T) {
	c := &client.Client{}

	tests := []struct {
		name    string
		data    any
		want    *client.Client
		wantErr bool
	}{
		{name: "nil data is not configured yet", data: nil},
		{name: "client", data: c, want: c},
		{name: "wrong type", data: "not a client", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := clientFromProviderData(tt.data, &diags)

			if got != tt.want {
				t.Errorf("got client %p, want %p", got, tt.want)
			}
			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError() = %v, want %v: %v", diags.HasError(), tt.wantErr, diags)
			}
			if tt.wantErr && !strings.Contains(diags[0].Detail(), "string") {
				t.Errorf("detail should name the type received, got %q", diags[0].Detail())
			}
		})
	}
}
