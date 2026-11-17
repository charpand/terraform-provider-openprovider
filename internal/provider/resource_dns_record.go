// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/dns"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &DNSRecordResource{}
	_ resource.ResourceWithConfigure   = &DNSRecordResource{}
	_ resource.ResourceWithImportState = &DNSRecordResource{}
)

// DNSRecordResource is the resource implementation.
type DNSRecordResource struct {
	client *client.Client
}

// DNSRecordModel describes the resource data model.
type DNSRecordModel struct {
	ZoneName         types.String `tfsdk:"zone_name"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	Value            types.String `tfsdk:"value"`
	TTL              types.Int64  `tfsdk:"ttl"`
	Priority         types.Int64  `tfsdk:"priority"`
	CreationDate     types.String `tfsdk:"creation_date"`
	ModificationDate types.String `tfsdk:"modification_date"`
	ID               types.String `tfsdk:"id"`
	AllowDeletion    types.Bool   `tfsdk:"allow_deletion"`
}

// NewDNSRecordResource returns a new instance of the DNS record resource.
func NewDNSRecordResource() resource.Resource {
	return &DNSRecordResource{}
}

// Metadata returns the resource type name.
func (r *DNSRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

// Schema defines the schema for the resource.
func (r *DNSRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS record in a zone.",
		Attributes: map[string]schema.Attribute{
			"zone_name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS zone containing this record (e.g., example.com).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS record (e.g., www, mail, @ for root).",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, SOA, etc.).",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the DNS record (IP address, hostname, or text).",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The time-to-live (TTL) in seconds for the record. Default is 3600.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "The priority for MX and SRV records. Lower values have higher priority.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the record was created.",
				Computed:            true,
			},
			"modification_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the record was last modified.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier for the DNS record (composite of zone_name, name, type, and value).",
				Computed:            true,
			},
			"allow_deletion": schema.BoolAttribute{
				MarkdownDescription: "Enable deletion of this DNS record. When false (default), the record is removed from Terraform state but preserved in OpenProvider. Set to true to permit actual deletion.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DNSRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSRecordModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := plan.ZoneName.ValueString()
	createReq := &dns.CreateRecordRequest{
		Name:     plan.Name.ValueString(),
		Type:     plan.Type.ValueString(),
		Value:    plan.Value.ValueString(),
		TTL:      int(plan.TTL.ValueInt64()),
		Priority: int(plan.Priority.ValueInt64()),
	}

	record, err := dns.CreateRecord(r.client, zoneName, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DNS record",
			fmt.Sprintf("Could not create DNS record: %s", err.Error()),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", zoneName, record.Name, record.Type))
	plan.CreationDate = types.StringValue(record.CreationDate)
	plan.ModificationDate = types.StringValue(record.ModificationDate)
	plan.TTL = types.Int64Value(int64(record.TTL))
	plan.Priority = types.Int64Value(int64(record.Priority))

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSRecordModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := state.ZoneName.ValueString()
	recordName := state.Name.ValueString()
	recordType := state.Type.ValueString()

	// A zone can hold several records of one name and type; the value tells
	// which one this resource manages. An import has no value yet.
	var record *dns.Record
	var err error
	if state.Value.IsNull() || state.Value.IsUnknown() {
		record, err = dns.GetRecord(r.client, zoneName, recordName, recordType)
	} else {
		record, err = dns.GetRecordWithValue(r.client, zoneName, recordName, recordType, state.Value.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DNS record",
			fmt.Sprintf("Could not read DNS record: %s", err.Error()),
		)
		return
	}

	// Update state
	state.Value = types.StringValue(record.Value)
	state.TTL = types.Int64Value(int64(record.TTL))
	state.Priority = types.Int64Value(int64(record.Priority))
	state.CreationDate = types.StringValue(record.CreationDate)
	state.ModificationDate = types.StringValue(record.ModificationDate)
	state.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", zoneName, record.Name, record.Type))

	// Set state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSRecordModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := plan.ZoneName.ValueString()
	updateReq := &dns.UpdateRecordRequest{
		Name:     plan.Name.ValueString(),
		Type:     plan.Type.ValueString(),
		Value:    plan.Value.ValueString(),
		TTL:      int(plan.TTL.ValueInt64()),
		Priority: int(plan.Priority.ValueInt64()),
	}

	record, err := dns.UpdateRecord(r.client, zoneName, plan.Name.ValueString(), plan.Type.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating DNS record",
			fmt.Sprintf("Could not update DNS record: %s", err.Error()),
		)
		return
	}

	// Update state
	plan.CreationDate = types.StringValue(record.CreationDate)
	plan.ModificationDate = types.StringValue(record.ModificationDate)
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", zoneName, record.Name, record.Type))

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource based on the allow_deletion flag.
func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSRecordModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := state.ZoneName.ValueString()
	recordName := state.Name.ValueString()
	recordType := state.Type.ValueString()
	recordValue := state.Value.ValueString()

	// Check if deletion is allowed
	allowDeletion := !state.AllowDeletion.IsNull() && state.AllowDeletion.ValueBool()

	if !allowDeletion {
		// Remove from state only - preserve the record in OpenProvider
		resp.Diagnostics.AddWarning(
			"DNS Record Removed from Terraform State Only",
			fmt.Sprintf("DNS record %s.%s (%s) has been removed from your Terraform state but NOT deleted in OpenProvider. "+
				"The record still exists and can be reimported. "+
				"To enable deletion, set allow_deletion = true on the resource.",
				recordName, zoneName, recordType),
		)
		return
	}

	// Proceed with deletion since allow_deletion is true
	err := dns.DeleteRecord(r.client, zoneName, recordName, recordType, recordValue)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DNS record",
			fmt.Sprintf("Could not delete DNS record: %s", err.Error()),
		)
		return
	}
}

// ImportState imports a record by `zone_name/name/type`, for example
// `example.com/www/A`. Where the zone has several records of this name and
// type, `zone_name/name/type/value` picks the one with this value. The record
// is imported with `allow_deletion` false, as a new record starts with it.
func (r *DNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" || (len(parts) == 4 && parts[3] == "") {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			fmt.Sprintf("Expected zone_name/name/type or zone_name/name/type/value (e.g. example.com/www/A), got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
	if len(parts) == 4 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), parts[3])...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_deletion"), false)...)
}
