// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NSGroupNameserverModel represents a nameserver in an NS group in Terraform state.
type NSGroupNameserverModel struct {
	Name types.String `tfsdk:"name"`
	IP   types.String `tfsdk:"ip"`
	IP6  types.String `tfsdk:"ip6"`
}

// NSGroupModel represents the Terraform state model for a nameserver group.
type NSGroupModel struct {
	ID          types.String             `tfsdk:"id"`
	Name        types.String             `tfsdk:"name"`
	Nameservers []NSGroupNameserverModel `tfsdk:"nameservers"`
}
