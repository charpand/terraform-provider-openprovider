// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DomainModel represents the Terraform state model for a domain.
// This is separate from the API model and uses Terraform framework types.
type DomainModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Status      types.String `tfsdk:"status"`
	Autorenew   types.Bool   `tfsdk:"autorenew"`
	Nameservers types.List   `tfsdk:"nameservers"`
}
