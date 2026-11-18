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
	_ resource.Resource              = &DomainRestoreResource{}
	_ resource.ResourceWithConfigure = &DomainRestoreResource{}
)

// DomainRestoreResource restores a domain that has expired when it is created
// or replaced. Its deletion does not change the domain.
type DomainRestoreResource struct {
	client *client.Client
}

// DomainRestoreModel is the Terraform state model for a restore.
type DomainRestoreModel struct {
	ID       types.String `tfsdk:"id"`
	Domain   types.String `tfsdk:"domain"`
	Triggers types.Map    `tfsdk:"triggers"`
	Status   types.String `tfsdk:"status"`
}

// NewDomainRestoreResource returns a new domain restore resource.
func NewDomainRestoreResource() resource.Resource {
	return &DomainRestoreResource{}
}

// Metadata sets the resource type name.
func (r *DomainRestoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_restore"
}

// Schema defines the resource schema.
func (r *DomainRestoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Restores a domain that has expired, where the registry and the API allow it, and charges the account for it " +
			"(`data.openprovider_domain_price` with `operation = \"restore\"` shows the price). " +
			"The restore occurs when Terraform creates or replaces the resource; a change of an attribute that the configuration sets replaces the resource, " +
			"so use `triggers` to restore again. The apply fails when the account has no such domain, or when the API refuses the restore, " +
			"for example because the domain is not expired or can no longer be recovered. The deletion of the resource does not change the domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name (e.g., example.com).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values. A change of a value restores the domain again.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status that OpenProvider answered the restore with.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure attaches the API client.
func (r *DomainRestoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Create restores the domain. A domain that has been deleted since it expired
// is not in the normal listing, so the lookup also asks for deleted domains.
func (r *DomainRestoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainRestoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Domain.ValueString()
	domain, err := getDomainByName(r.client, name)
	if err == nil && domain == nil {
		domain, err = getDeletedDomainByName(r.client, name)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Domain", fmt.Sprintf("Could not read domain %s: %s", name, err))
		return
	}
	if domain == nil {
		resp.Diagnostics.AddError("Domain Not Found", fmt.Sprintf("The account has no domain %s, so there is nothing to restore.", name))
		return
	}

	status, err := domains.Restore(r.client, domain.ID, domain.Domain.Name, domain.Domain.Extension)
	if err != nil {
		resp.Diagnostics.AddError("Error Restoring Domain", fmt.Sprintf("Could not restore domain %s: %s", name, err))
		return
	}

	plan.ID = plan.Domain
	plan.Status = types.StringValue(status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read keeps the state. A restore is an action, not an object at OpenProvider
// that can be read back.
func (r *DomainRestoreResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Update keeps the plan. Each attribute that the configuration sets replaces
// the resource on a change, so an update does not restore the domain.
func (r *DomainRestoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainRestoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the resource from the state only.
func (r *DomainRestoreResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
