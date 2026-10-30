// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &DomainAuthCodeResetResource{}
	_ resource.ResourceWithConfigure = &DomainAuthCodeResetResource{}
)

// DomainAuthCodeResetResource replaces the auth code of a domain with a new
// code when it is created or replaced, so the old code no longer transfers
// the domain. It does not keep the new code. Its deletion does not change the
// domain.
type DomainAuthCodeResetResource struct {
	client *client.Client
}

// DomainAuthCodeResetModel is the Terraform state model for a reset of the
// auth code of a domain.
type DomainAuthCodeResetModel struct {
	ID           types.String `tfsdk:"id"`
	Domain       types.String `tfsdk:"domain"`
	AuthCodeType types.String `tfsdk:"auth_code_type"`
	Triggers     types.Map    `tfsdk:"triggers"`
}

// NewDomainAuthCodeResetResource returns a new domain auth code reset resource.
func NewDomainAuthCodeResetResource() resource.Resource {
	return &DomainAuthCodeResetResource{}
}

// Metadata sets the resource type name.
func (r *DomainAuthCodeResetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_auth_code_reset"
}

// Schema defines the resource schema.
func (r *DomainAuthCodeResetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Replaces the auth code of a domain that the account holds with a new code, so the old code no longer transfers the domain. The reset occurs when Terraform creates or replaces the resource. A change of `triggers` replaces the resource. The resource does not keep the new code; use the `openprovider_domain_auth_code` ephemeral resource to read it. The deletion of the resource does not change the domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name (e.g., example.com).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auth_code_type": schema.StringAttribute{
				MarkdownDescription: authCodeTypeDescription,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(domains.AuthCodeTypeExternal),
				Validators:          []validator.String{authCodeTypeValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values. A change of a value resets the auth code again.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure attaches the API client.
func (r *DomainAuthCodeResetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create resets the auth code.
func (r *DomainAuthCodeResetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainAuthCodeResetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := heldDomainID(r.client, plan.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	if _, err := domains.ResetAuthCode(r.client, id, authCodeTypeOf(plan.AuthCodeType)); err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Auth Code",
			fmt.Sprintf("Could not reset the auth code of domain %s: %s", plan.Domain.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read keeps the state. A reset is an action, not an object at OpenProvider
// that can be read back.
func (r *DomainAuthCodeResetResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Update keeps the plan. Each attribute that the configuration sets replaces
// the resource on a change, so an update does not reset the auth code.
func (r *DomainAuthCodeResetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainAuthCodeResetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the resource from the state only. The domain keeps its auth
// code.
func (r *DomainAuthCodeResetResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
