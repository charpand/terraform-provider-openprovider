package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &DomainRenewalResource{}
	_ resource.ResourceWithConfigure = &DomainRenewalResource{}
)

// maxRenewalPeriod is the longest period of one renewal, in years.
const maxRenewalPeriod = 10

// DomainRenewalResource renews a domain when it is created or replaced. Its
// deletion does not change the domain.
type DomainRenewalResource struct {
	client *client.Client
}

// DomainRenewalModel is the Terraform state model for a renewal.
type DomainRenewalModel struct {
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	Period         types.Int64  `tfsdk:"period"`
	Triggers       types.Map    `tfsdk:"triggers"`
	Status         types.String `tfsdk:"status"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

// NewDomainRenewalResource returns a new domain renewal resource.
func NewDomainRenewalResource() resource.Resource {
	return &DomainRenewalResource{}
}

// Metadata sets the resource type name.
func (r *DomainRenewalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_renewal"
}

// Schema defines the resource schema.
func (r *DomainRenewalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Renews a domain for a period of years, and charges the account for it. " +
			"The renewal occurs when Terraform creates or replaces the resource; a change of an attribute that the configuration sets replaces the resource, " +
			"so use `triggers` to renew again. The apply fails, and the provider sends no renewal, when the account has no such domain, " +
			"when the domain is not active, or when OpenProvider does not allow a renewal of the domain now (see `can_renew` of the `openprovider_domains` data source). " +
			"The deletion of the resource does not change the domain.",
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
			"period": schema.Int64Attribute{
				MarkdownDescription: "The period of the renewal in years, from 1 to 10. Defaults to 1. The registry limits the total registration period of the domain.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values. A change of a value renews the domain again.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status that OpenProvider answered the renewal with.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "When the registration expires, as read after the renewal. The registry can take a while to move it.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure attaches the API client.
func (r *DomainRenewalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Create renews the domain.
func (r *DomainRenewalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainRenewalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Domain.ValueString()
	period := plan.Period.ValueInt64()
	if period < 1 || period > maxRenewalPeriod {
		resp.Diagnostics.AddAttributeError(path.Root("period"), "Invalid Period",
			fmt.Sprintf("The period is %d years, it must be between 1 and %d.", period, maxRenewalPeriod))
		return
	}

	domain, err := getDomainByName(r.client, name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Domain", fmt.Sprintf("Could not read domain %s: %s", name, err))
		return
	}
	if domain == nil {
		resp.Diagnostics.AddError("Domain Not Found", fmt.Sprintf("The account has no domain %s, so there is nothing to renew.", name))
		return
	}
	if domain.Status != domainStatusActive {
		resp.Diagnostics.AddError("Domain Is Not Active",
			fmt.Sprintf("Domain %s has status %s, and only a domain with status %s can be renewed.", name, domain.Status, domainStatusActive))
		return
	}
	if !domain.CanRenew {
		resp.Diagnostics.AddError("Renewal Not Allowed",
			fmt.Sprintf("OpenProvider does not allow a renewal of domain %s now. It expires on %s.", name, domain.ExpirationDate))
		return
	}

	status, err := domains.Renew(r.client, domain.ID, domain.Domain.Name, domain.Domain.Extension, int(period))
	if err != nil {
		resp.Diagnostics.AddError("Error Renewing Domain", fmt.Sprintf("Could not renew domain %s: %s", name, err))
		return
	}

	plan.ID = plan.Domain
	plan.Status = types.StringValue(status)
	plan.ExpirationDate = types.StringNull()
	// The renewal is done. A failed read after it only leaves the date empty.
	if renewed, err := getDomainByName(r.client, name); err == nil && renewed != nil && renewed.ExpirationDate != "" {
		plan.ExpirationDate = types.StringValue(renewed.ExpirationDate)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read keeps the state. A renewal is an action, not an object at OpenProvider
// that can be read back.
func (r *DomainRenewalResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Update keeps the plan. Each attribute that the configuration sets replaces
// the resource on a change, so an update does not renew the domain.
func (r *DomainRenewalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainRenewalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the resource from the state only.
func (r *DomainRenewalResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
