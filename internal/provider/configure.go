package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// clientFromProviderData returns the API client that the provider hands to its
// resources, data sources and ephemeral resources in `Configure`. It returns
// nil without an error while the provider is not configured yet, which is when
// the framework calls `Configure` with nil data, and nil with an error for
// data of any other type.
func clientFromProviderData(providerData any, diags *diag.Diagnostics) *client.Client {
	if providerData == nil {
		return nil
	}

	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return c
}
