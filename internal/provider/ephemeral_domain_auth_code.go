// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ ephemeral.EphemeralResource              = &DomainAuthCodeEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &DomainAuthCodeEphemeralResource{}
)

// DomainAuthCodeEphemeralResource reads the auth code of a domain that the
// account holds. The code transfers the domain out of the account, so it is
// ephemeral: Terraform does not keep it in the plan or the state.
type DomainAuthCodeEphemeralResource struct {
	client *client.Client
}

// DomainAuthCodeModel is the Terraform model for the auth code of a domain.
type DomainAuthCodeModel struct {
	Domain       types.String `tfsdk:"domain"`
	AuthCodeType types.String `tfsdk:"auth_code_type"`
	AuthCode     types.String `tfsdk:"auth_code"`
}

// NewDomainAuthCodeEphemeralResource returns a new domain auth code ephemeral
// resource.
func NewDomainAuthCodeEphemeralResource() ephemeral.EphemeralResource {
	return &DomainAuthCodeEphemeralResource{}
}

// Metadata sets the ephemeral resource type name.
func (r *DomainAuthCodeEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_auth_code"
}

// Schema defines the ephemeral resource schema.
func (r *DomainAuthCodeEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the auth code of a domain that the account holds. The auth code lets a different account transfer the domain. Terraform does not keep it in the plan or the state.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name (e.g., example.com).",
				Required:            true,
			},
			"auth_code_type": schema.StringAttribute{
				MarkdownDescription: authCodeTypeDescription,
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{authCodeTypeValidator{}},
			},
			"auth_code": schema.StringAttribute{
				MarkdownDescription: "The auth code of the domain.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure attaches the API client.
func (r *DomainAuthCodeEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Open reads the auth code.
func (r *DomainAuthCodeEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data DomainAuthCodeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := heldDomainID(r.client, data.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	authCodeType := authCodeTypeOf(data.AuthCodeType)
	code, err := domains.GetAuthCode(r.client, id, authCodeType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Auth Code",
			fmt.Sprintf("Could not read the auth code of domain %s: %s", data.Domain.ValueString(), err.Error()),
		)
		return
	}
	if code.AuthCode == "" {
		resp.Diagnostics.AddError(
			"Error Reading Auth Code",
			fmt.Sprintf("OpenProvider gave no auth code for domain %s.", data.Domain.ValueString()),
		)
		return
	}

	data.AuthCodeType = types.StringValue(authCodeType)
	data.AuthCode = types.StringValue(code.AuthCode)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

const authCodeTypeDescription = "The type of the auth code: `external` (the default) to transfer the domain to a different registrar, or `internal` to transfer the domain to a different reseller account at OpenProvider."

// authCodeTypeOf gives the auth code type that a configuration sets, or
// `external` when it does not set one.
func authCodeTypeOf(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return domains.AuthCodeTypeExternal
	}
	return v.ValueString()
}

// authCodeTypeValidator accepts only the auth code types of OpenProvider.
type authCodeTypeValidator struct{}

// Description describes the validation.
func (v authCodeTypeValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be %q or %q", domains.AuthCodeTypeExternal, domains.AuthCodeTypeInternal)
}

// MarkdownDescription describes the validation in Markdown.
func (v authCodeTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks the value.
func (v authCodeTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	switch req.ConfigValue.ValueString() {
	case domains.AuthCodeTypeExternal, domains.AuthCodeTypeInternal:
	default:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Auth Code Type", v.Description(ctx)+".")
	}
}

// heldDomainID gives the id of a domain that the account holds. When the
// account does not hold the domain, or the lookup fails, it adds an error to
// `diags` and gives false.
func heldDomainID(c *client.Client, name string, diags *diag.Diagnostics) (int, bool) {
	domain, err := getDomainByName(c, name)
	if err != nil {
		diags.AddError(
			"Error Reading Domain",
			fmt.Sprintf("Could not find domain %s: %s", name, err.Error()),
		)
		return 0, false
	}
	if domain == nil {
		diags.AddError(
			"Domain Not Found",
			fmt.Sprintf("The account does not hold domain %s.", name),
		)
		return 0, false
	}
	return domain.ID, true
}
