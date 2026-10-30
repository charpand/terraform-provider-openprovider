// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DomainModel represents the Terraform state model for a domain.
// This is separate from the API model and uses Terraform framework types.
type DomainModel struct {
	ID              types.String `tfsdk:"id"`
	Domain          types.String `tfsdk:"domain"`
	AuthCode        types.String `tfsdk:"auth_code"`
	Status          types.String `tfsdk:"status"`
	Autorenew       types.Bool   `tfsdk:"autorenew"`
	OnDestroy       types.String `tfsdk:"on_destroy"`
	OwnerHandle     types.String `tfsdk:"owner_handle"`
	AdminHandle     types.String `tfsdk:"admin_handle"`
	TechHandle      types.String `tfsdk:"tech_handle"`
	BillingHandle   types.String `tfsdk:"billing_handle"`
	Period          types.Int64  `tfsdk:"period"`
	MaxCost         types.Int64  `tfsdk:"max_cost"`
	Currency        types.String `tfsdk:"currency"`
	Cost            types.Int64  `tfsdk:"cost"`
	NSGroup         types.String `tfsdk:"ns_group"`
	DnssecKeys      types.List   `tfsdk:"dnssec_keys"`
	IsDnssecEnabled types.Bool   `tfsdk:"is_dnssec_enabled"`
	IsLocked        types.Bool   `tfsdk:"is_locked"`
	ExpirationDate  types.String `tfsdk:"expiration_date"`
}

// DnssecKeyModel represents a DNSSEC key in Terraform state.
type DnssecKeyModel struct {
	Algorithm types.Int64  `tfsdk:"algorithm"`
	Flags     types.Int64  `tfsdk:"flags"`
	Protocol  types.Int64  `tfsdk:"protocol"`
	PublicKey types.String `tfsdk:"public_key"`
}

// DomainCheckModel describes the data source data model for a domain
// availability check.
type DomainCheckModel struct {
	ID        types.String `tfsdk:"id"`
	Domain    types.String `tfsdk:"domain"`
	Status    types.String `tfsdk:"status"`
	Available types.Bool   `tfsdk:"available"`
	IsPremium types.Bool   `tfsdk:"is_premium"`
	Reason    types.String `tfsdk:"reason"`
}

// DomainsModel describes the data source data model for a domain listing.
type DomainsModel struct {
	ID       types.String        `tfsdk:"id"`
	FullName types.String        `tfsdk:"full_name"`
	Domains  []DomainsEntryModel `tfsdk:"domains"`
}

// DomainsEntryModel is one domain in a listing.
type DomainsEntryModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	Status         types.String `tfsdk:"status"`
	OwnerHandle    types.String `tfsdk:"owner_handle"`
	NSGroup        types.String `tfsdk:"ns_group"`
	Autorenew      types.Bool   `tfsdk:"autorenew"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}
