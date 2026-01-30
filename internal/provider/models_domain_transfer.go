// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DomainTransferModel represents the Terraform state model for a domain transfer.
type DomainTransferModel struct {
	ID                            types.String `tfsdk:"id"`
	Domain                        types.String `tfsdk:"domain"`
	AuthCode                      types.String `tfsdk:"auth_code"`
	Status                        types.String `tfsdk:"status"`
	Autorenew                     types.Bool   `tfsdk:"autorenew"`
	OwnerHandle                   types.String `tfsdk:"owner_handle"`
	AdminHandle                   types.String `tfsdk:"admin_handle"`
	TechHandle                    types.String `tfsdk:"tech_handle"`
	BillingHandle                 types.String `tfsdk:"billing_handle"`
	NSGroup                       types.String `tfsdk:"ns_group"`
	ImportContactsFromRegistry    types.Bool   `tfsdk:"import_contacts_from_registry"`
	ImportNameserversFromRegistry types.Bool   `tfsdk:"import_nameservers_from_registry"`
	IsPrivateWhoisEnabled         types.Bool   `tfsdk:"is_private_whois_enabled"`
	ExpirationDate                types.String `tfsdk:"expiration_date"`
}
