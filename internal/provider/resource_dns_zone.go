// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/dns"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &DNSZoneResource{}
	_ resource.ResourceWithConfigure   = &DNSZoneResource{}
	_ resource.ResourceWithImportState = &DNSZoneResource{}
)

// DNSZoneResource is the resource implementation.
type DNSZoneResource struct {
	client *client.Client
}

// DNSZoneResourceModel describes the resource data model.
type DNSZoneResourceModel struct {
	ZoneName         types.String `tfsdk:"zone_name"`
	Type             types.String `tfsdk:"type"`
	Provider         types.String `tfsdk:"provider_name"`
	Secured          types.Bool   `tfsdk:"secured"`
	MasterIP         types.String `tfsdk:"master_ip"`
	TemplateName     types.String `tfsdk:"template_name"`
	AllowDeletion    types.Bool   `tfsdk:"allow_deletion"`
	Name             types.String `tfsdk:"name"`
	Extension        types.String `tfsdk:"extension"`
	CreationDate     types.String `tfsdk:"creation_date"`
	ModificationDate types.String `tfsdk:"modification_date"`
	ID               types.String `tfsdk:"id"`
}

// NewDNSZoneResource returns a new instance of the DNS zone resource.
func NewDNSZoneResource() resource.Resource {
	return &DNSZoneResource{}
}

// Metadata returns the resource type name.
func (r *DNSZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

// Schema defines the schema for the resource.
func (r *DNSZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS zone. A zone holds the records that `openprovider_dns_record` manages. " +
			"A change of any argument replaces the zone, and with it all of its records.",
		Attributes: map[string]schema.Attribute{
			"zone_name": schema.StringAttribute{
				MarkdownDescription: "The name of the zone, for example `example.com`.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the zone: `master` (default) or `slave`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("master"),
				PlanModifiers:       replace,
			},
			"provider_name": schema.StringAttribute{
				MarkdownDescription: "The DNS provider that hosts the zone. If it is not set, OpenProvider uses its default (`openprovider`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"secured": schema.BoolAttribute{
				MarkdownDescription: "Whether the zone is signed with DNSSEC. If it is not set, OpenProvider decides.",
				Optional:            true,
			},
			"master_ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the master name server. Only for a zone of the type `slave`.",
				Optional:            true,
				PlanModifiers:       replace,
			},
			"template_name": schema.StringAttribute{
				MarkdownDescription: "The name of a DNS template (see `openprovider_dns_template`) whose records the new zone starts with. Only used when the zone is created.",
				Optional:            true,
				PlanModifiers:       replace,
			},
			"allow_deletion": schema.BoolAttribute{
				MarkdownDescription: "Enable deletion of this zone. When false (default), the zone is removed from Terraform state but preserved in OpenProvider, as for `openprovider_dns_record`. Set to true to permit actual deletion, which also deletes all records of the zone.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name part of the zone (`example` in `example.com`).",
				Computed:            true,
			},
			"extension": schema.StringAttribute{
				MarkdownDescription: "The extension of the zone (`com` in `example.com`).",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the zone was created.",
				Computed:            true,
			},
			"modification_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the zone was last modified.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The zone name.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DNSZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create creates the zone and sets the initial Terraform state.
func (r *DNSZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := plan.ZoneName.ValueString()
	name, extension, ok := domains.SplitFullName(zoneName)
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("zone_name"), "Invalid zone name",
			fmt.Sprintf("%q is not a zone name such as example.com.", zoneName))
		return
	}

	createReq := &dns.CreateZoneRequest{
		Domain:       dns.ZoneDomain{Name: name, Extension: extension},
		Type:         plan.Type.ValueString(),
		Provider:     plan.Provider.ValueString(),
		MasterIP:     plan.MasterIP.ValueString(),
		TemplateName: plan.TemplateName.ValueString(),
	}
	if !plan.Secured.IsNull() && !plan.Secured.IsUnknown() {
		secured := plan.Secured.ValueBool()
		createReq.Secured = &secured
	}

	if err := dns.CreateZone(r.client, createReq); err != nil {
		resp.Diagnostics.AddError("Error creating DNS zone", fmt.Sprintf("Could not create DNS zone: %s", err))
		return
	}

	zone, err := dns.GetZone(r.client, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Error reading DNS zone", fmt.Sprintf("The zone was created, but reading it failed: %s", err))
		return
	}

	r.applyZone(&plan, zone)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DNSZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := dns.GetZone(r.client, state.ZoneName.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading DNS zone", fmt.Sprintf("Could not read DNS zone: %s", err))
		return
	}

	r.applyZone(&state, zone)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update only changes arguments that do not call the API: `allow_deletion` and, as the API
// has no call to change it, nothing else. Every other argument replaces the zone.
func (r *DNSZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the zone, when `allow_deletion` is true.
func (r *DNSZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := state.ZoneName.ValueString()
	if state.AllowDeletion.IsNull() || !state.AllowDeletion.ValueBool() {
		resp.Diagnostics.AddWarning(
			"DNS Zone Removed from Terraform State Only",
			fmt.Sprintf("DNS zone %s has been removed from your Terraform state but NOT deleted in OpenProvider. "+
				"The zone and its records still exist and can be imported again. "+
				"To enable deletion, set allow_deletion = true on the resource.", zoneName),
		)
		return
	}

	if err := dns.DeleteZone(r.client, zoneName); err != nil {
		resp.Diagnostics.AddError("Error deleting DNS zone", fmt.Sprintf("Could not delete DNS zone: %s", err))
	}
}

// ImportState imports a zone by its name, for example `example.com`.
func (r *DNSZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_deletion"), false)...)
}

func (r *DNSZoneResource) applyZone(m *DNSZoneResourceModel, zone *dns.Zone) {
	m.Name = types.StringValue(zone.Name)
	m.Extension = types.StringValue(zone.Extension)
	if zone.Type != "" {
		m.Type = types.StringValue(zone.Type)
	}
	m.CreationDate = types.StringValue(zone.CreationDate)
	m.ModificationDate = types.StringValue(zone.ModificationDate)
	m.ID = types.StringValue(m.ZoneName.ValueString())
	if m.Provider.IsUnknown() || m.Provider.IsNull() {
		m.Provider = types.StringValue(zone.Provider)
	}
}
