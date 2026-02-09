// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DomainModel represents the Terraform state model for a domain.
// This is separate from the API model and uses Terraform framework types.
type DomainModel struct {
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	AuthCode       types.String `tfsdk:"auth_code"`
	Status         types.String `tfsdk:"status"`
	Autorenew      types.Bool   `tfsdk:"autorenew"`
	OwnerHandle    types.String `tfsdk:"owner_handle"`
	AdminHandle    types.String `tfsdk:"admin_handle"`
	TechHandle     types.String `tfsdk:"tech_handle"`
	BillingHandle  types.String `tfsdk:"billing_handle"`
	Period         types.Int64  `tfsdk:"period"`
	NSGroup        types.String `tfsdk:"ns_group"`
	DSRecords      types.List   `tfsdk:"ds_records"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

// DSRecordModel represents a DS record in Terraform state.
type DSRecordModel struct {
	Algorithm types.Int64  `tfsdk:"algorithm"`
	Flags     types.Int64  `tfsdk:"flags"`
	Protocol  types.Int64  `tfsdk:"protocol"`
	PublicKey types.String `tfsdk:"public_key"`
}
