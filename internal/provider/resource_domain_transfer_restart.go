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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &DomainTransferRestartResource{}
	_ resource.ResourceWithConfigure = &DomainTransferRestartResource{}
)

// DomainTransferRestartResource sends a failed transfer to the registry again
// when it is created or replaced, without a new order. Its deletion does not
// change the domain.
type DomainTransferRestartResource struct {
	client *client.Client
}

// DomainTransferRestartModel is the Terraform state model for a restart of a
// failed transfer.
type DomainTransferRestartModel struct {
	ID       types.String `tfsdk:"id"`
	Domain   types.String `tfsdk:"domain"`
	AuthCode types.String `tfsdk:"auth_code"`
	Triggers types.Map    `tfsdk:"triggers"`
}

// NewDomainTransferRestartResource returns a new domain transfer restart resource.
func NewDomainTransferRestartResource() resource.Resource {
	return &DomainTransferRestartResource{}
}

// Metadata sets the resource type name.
func (r *DomainTransferRestartResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_transfer_restart"
}

// Schema defines the resource schema.
func (r *DomainTransferRestartResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sends a transfer that failed to the registry again, without a new order. The restart occurs when Terraform creates or replaces the resource. A change of an attribute that the configuration sets replaces the resource. The domain must have the status `FAI`. With a different status, or when the account has no such domain, the apply fails and the provider sends no restart. The deletion of the resource does not change the domain.",
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
			"auth_code": schema.StringAttribute{
				MarkdownDescription: "The auth code to send with the restart. Terraform keeps this value in the state.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values. A change of a value restarts the transfer again.",
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
func (r *DomainTransferRestartResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create restarts the transfer. The API restarts the last operation of the
// domain, and this operation is not always a transfer, so only a domain with
// a failed operation gets a restart.
func (r *DomainTransferRestartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainTransferRestartModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Domain.ValueString()
	domain, err := getDomainByName(r.client, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain",
			fmt.Sprintf("Could not read domain %s: %s", name, err.Error()),
		)
		return
	}
	if domain == nil {
		resp.Diagnostics.AddError(
			"Domain Not Found",
			fmt.Sprintf("The account has no domain %s, so there is no transfer to restart.", name),
		)
		return
	}
	if domain.Status != domainStatusFailed {
		resp.Diagnostics.AddError(
			"Domain Transfer Has Not Failed",
			fmt.Sprintf("Domain %s has status %s, and only a domain with status %s can restart its transfer.", name, domain.Status, domainStatusFailed),
		)
		return
	}

	if err := domains.RestartLastOperation(r.client, domain.ID, domain.Domain.Name, domain.Domain.Extension, plan.AuthCode.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Restarting Domain Transfer",
			fmt.Sprintf("Could not restart the transfer of domain %s: %s", name, err.Error()),
		)
		return
	}

	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read keeps the state. A restart is an action, not an object at OpenProvider
// that can be read back.
func (r *DomainTransferRestartResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Update keeps the plan. Each attribute that the configuration sets replaces
// the resource on a change, so an update does not restart the transfer.
func (r *DomainTransferRestartResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainTransferRestartModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the resource from the state only.
func (r *DomainTransferRestartResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
