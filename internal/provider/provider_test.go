package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Schema attributes should not be nil")
	}

	expectedAttrs := []string{"username", "password"}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Expected attribute %s not found in schema", attr)
		}
	}
}
