// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/dns"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &DNSTemplateResource{}
	_ resource.ResourceWithConfigure   = &DNSTemplateResource{}
	_ resource.ResourceWithImportState = &DNSTemplateResource{}
)

// DNSTemplateResource is the resource implementation.
type DNSTemplateResource struct {
	client *client.Client
}

// DNSTemplateModel describes the resource data model.
type DNSTemplateModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	IsSpamexpertsEnabled types.Bool   `tfsdk:"is_spamexperts_enabled"`
	Records              types.Set    `tfsdk:"records"`
}

// DNSTemplateRecordModel is one record of a template.
type DNSTemplateRecordModel struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
}

var dnsTemplateRecordAttrTypes = map[string]attr.Type{
	"name":     types.StringType,
	"type":     types.StringType,
	"value":    types.StringType,
	"ttl":      types.Int64Type,
	"priority": types.Int64Type,
}

// NewDNSTemplateResource returns a new instance of the DNS template resource.
func NewDNSTemplateResource() resource.Resource {
	return &DNSTemplateResource{}
}

// Metadata returns the resource type name.
func (r *DNSTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_template"
}

// Schema defines the schema for the resource.
func (r *DNSTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS template: a set of records that a new zone can start with " +
			"(`template_name` of `openprovider_dns_zone`). The API has no call to change a template, " +
			"so a change of any argument replaces the template. A zone that was created from a template keeps its records when the template is replaced or deleted.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the template at OpenProvider.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the template.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_spamexperts_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether a zone from this template uses SpamExperts. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"records": schema.SetNestedAttribute{
				MarkdownDescription: "The records of the template. A record name is relative to the zone (for example `www`, or empty for the zone itself).",
				Required:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the record, relative to the zone.",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The record type (A, AAAA, CNAME, MX, TXT, ...).",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value of the record.",
							Required:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "The time-to-live in seconds.",
							Required:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "The priority of an MX or SRV record. Defaults to 0.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DNSTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the template and sets the initial Terraform state.
func (r *DNSTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var records []DNSTemplateRecordModel
	resp.Diagnostics.Append(plan.Records.ElementsAs(ctx, &records, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &dns.CreateTemplateRequest{
		Name:                 plan.Name.ValueString(),
		IsSpamexpertsEnabled: plan.IsSpamexpertsEnabled.ValueBool(),
		Records:              make([]dns.TemplateRecord, 0, len(records)),
	}
	for _, record := range records {
		createReq.Records = append(createReq.Records, dns.TemplateRecord{
			Name:     record.Name.ValueString(),
			Type:     record.Type.ValueString(),
			Value:    record.Value.ValueString(),
			TTL:      int(record.TTL.ValueInt64()),
			Priority: int(record.Priority.ValueInt64()),
		})
	}

	id, err := dns.CreateTemplate(r.client, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS template", fmt.Sprintf("Could not create DNS template: %s", err))
		return
	}

	// The state keeps what the plan holds, so that a different spelling by the
	// API does not turn into a difference after the create.
	plan.ID = types.StringValue(strconv.Itoa(id))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DNSTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid DNS template id", fmt.Sprintf("The id %q is not a number.", state.ID.ValueString()))
		return
	}

	template, err := dns.GetTemplate(r.client, id)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading DNS template", fmt.Sprintf("Could not read DNS template: %s", err))
		return
	}
	if template.ID == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	records, diags := templateRecordsToSet(ctx, template.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Name = types.StringValue(template.Name)
	state.IsSpamexpertsEnabled = types.BoolValue(template.IsSpamexpertsEnabled)
	state.Records = records
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update keeps the plan: every argument replaces the template on a change, so
// an update has nothing to send.
func (r *DNSTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the template. A zone that was created from it keeps its records.
func (r *DNSTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid DNS template id", fmt.Sprintf("The id %q is not a number.", state.ID.ValueString()))
		return
	}

	if err := dns.DeleteTemplate(r.client, id); err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting DNS template", fmt.Sprintf("Could not delete DNS template: %s", err))
	}
}

// ImportState imports a template by its id.
func (r *DNSTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import Identifier", fmt.Sprintf("Expected the numeric id of a DNS template, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func templateRecordsToSet(ctx context.Context, records []dns.TemplateRecord) (types.Set, diag.Diagnostics) {
	items := make([]DNSTemplateRecordModel, 0, len(records))
	for _, record := range records {
		items = append(items, DNSTemplateRecordModel{
			Name:     types.StringValue(record.Name),
			Type:     types.StringValue(record.Type),
			Value:    types.StringValue(record.Value),
			TTL:      types.Int64Value(int64(record.TTL)),
			Priority: types.Int64Value(int64(record.Priority)),
		})
	}
	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: dnsTemplateRecordAttrTypes}, items)
}
