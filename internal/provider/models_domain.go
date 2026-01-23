// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NameserverModel represents a nameserver in Terraform state.
type NameserverModel struct {
	Hostname types.String `tfsdk:"hostname"`
}

// DomainModel represents the Terraform state model for a domain.
// This is separate from the API model and uses Terraform framework types.
type DomainModel struct {
	ID            types.String      `tfsdk:"id"`
	Name          types.String      `tfsdk:"name"`
	Status        types.String      `tfsdk:"status"`
	Autorenew     types.Bool        `tfsdk:"autorenew"`
	OwnerHandle   types.String      `tfsdk:"owner_handle"`
	AdminHandle   types.String      `tfsdk:"admin_handle"`
	TechHandle    types.String      `tfsdk:"tech_handle"`
	BillingHandle types.String      `tfsdk:"billing_handle"`
	Period        types.Int64       `tfsdk:"period"`
	Nameservers   []NameserverModel `tfsdk:"nameserver"`
}
