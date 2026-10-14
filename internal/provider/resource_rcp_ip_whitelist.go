// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/contacts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The value `on_destroy` takes besides `retain` on the whitelist resource.
const onDestroyClear = "clear"

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &RCPIPWhitelistResource{}
	_ resource.ResourceWithConfigure   = &RCPIPWhitelistResource{}
	_ resource.ResourceWithImportState = &RCPIPWhitelistResource{}
)

// RCPIPWhitelistResourceModel is the Terraform state of the whitelist.
type RCPIPWhitelistResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ContactID types.Int64  `tfsdk:"contact_id"`
	Allow     types.Set    `tfsdk:"allow"`
	OnDestroy types.String `tfsdk:"on_destroy"`
}

// RCPIPWhitelistResource manages the control panel IP whitelist of a contact.
type RCPIPWhitelistResource struct {
	client *client.Client
}

// NewRCPIPWhitelistResource returns a new instance of the whitelist resource.
func NewRCPIPWhitelistResource() resource.Resource {
	return &RCPIPWhitelistResource{}
}

// Metadata returns the resource type name.
func (r *RCPIPWhitelistResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rcp_ip_whitelist"
}

// Schema defines the schema for the resource.
func (r *RCPIPWhitelistResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the IP whitelist for the OpenProvider control panel (RCP) of one contact. " +
			"The resource owns the contact's whole `allow` list: whatever the list holds beyond `allow` is removed on apply. " +
			"The contact's `deny` list and its other settings are not touched.\n\n" +
			"~> **Warning:** a whitelist locks out every address that is not on it, and the API answers them with error 10005 (Access Denied). " +
			"Include the address you apply from, and any CI runner that applies this configuration, or the next apply may be refused. " +
			"Destroying the resource with `on_destroy = \"clear\"` switches the whitelist off.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The contact identifier, the same as `contact_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contact_id": schema.Int64Attribute{
				MarkdownDescription: "The numeric id of the contact whose control panel access is restricted (the login contact, not a customer handle). Changing it replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"allow": schema.SetAttribute{
				MarkdownDescription: "The addresses allowed to reach the control panel: IPv4 or IPv6 addresses, or CIDR blocks. At least one entry is required, since an empty list switches the whitelist off. Entries are compared as written, so write an address the way the API returns it.",
				Required:            true,
				ElementType:         types.StringType,
				Validators:          []validator.Set{ipListValidator{}},
			},
			"on_destroy": schema.StringAttribute{
				MarkdownDescription: "What destroying this resource does at OpenProvider. `retain` (the default) only forgets the whitelist in state: it stays in force at OpenProvider. `clear` empties the list, which **switches the whitelist off** and opens the control panel to every address.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(onDestroyRetain),
				Validators: []validator.String{
					oneOfValidator{allowed: []string{onDestroyRetain, onDestroyClear}},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RCPIPWhitelistResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// allowStrings returns the entries of the plan's set, sorted.
func allowStrings(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	sort.Strings(out)
	return out, diags
}

// allowSet builds a state set from the list the API reports.
func allowSet(allow []string) types.Set {
	values := make([]string, len(allow))
	copy(values, allow)
	set, _ := types.SetValueFrom(context.Background(), types.StringType, values)
	return set
}

// apply writes the planned list and reads it back, so state holds what the
// API holds rather than what was sent.
func (r *RCPIPWhitelistResource) apply(ctx context.Context, plan *RCPIPWhitelistResourceModel, action string) diag.Diagnostics {
	var diags diag.Diagnostics
	id := int(plan.ContactID.ValueInt64())

	allow, d := allowStrings(ctx, plan.Allow)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	if err := contacts.SetRCPAllowList(r.client, id, allow); err != nil {
		diags.AddError(
			"Error "+action+" RCP IP Whitelist",
			fmt.Sprintf("Could not set the whitelist of contact %d: %s", id, err.Error()),
		)
		return diags
	}

	contact, err := contacts.Get(r.client, id)
	if err != nil {
		diags.AddError(
			"Error Reading RCP IP Whitelist",
			fmt.Sprintf("Set the whitelist of contact %d but could not read it back: %s", id, err.Error()),
		)
		return diags
	}
	if contact == nil {
		diags.AddError(
			"Error Reading RCP IP Whitelist",
			fmt.Sprintf("Set the whitelist of contact %d but the API does not report the contact.", id),
		)
		return diags
	}

	plan.ID = types.StringValue(strconv.Itoa(id))
	plan.Allow = allowSet(reportedAllow(contact))
	return diags
}

// reportedAllow is the allow list a contact carries; a contact without one
// has an empty list.
func reportedAllow(contact *contacts.Contact) []string {
	if contact.RCPClientIPList == nil {
		return nil
	}
	return contact.RCPClientIPList.Allow
}

// Create sets the whitelist and the initial Terraform state.
func (r *RCPIPWhitelistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RCPIPWhitelistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.apply(ctx, &plan, "Creating")...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state; a list that differs from state shows
// up as drift in the next plan.
func (r *RCPIPWhitelistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RCPIPWhitelistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ContactID.ValueInt64())
	contact, err := contacts.Get(r.client, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading RCP IP Whitelist",
			fmt.Sprintf("Could not read contact %d: %s", id, err.Error()),
		)
		return
	}
	if contact == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(strconv.Itoa(id))
	state.Allow = allowSet(reportedAllow(contact))
	if state.OnDestroy.IsNull() || state.OnDestroy.IsUnknown() {
		state.OnDestroy = types.StringValue(onDestroyRetain)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update replaces the whitelist with the planned one.
func (r *RCPIPWhitelistResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RCPIPWhitelistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.apply(ctx, &plan, "Updating")...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete does to the whitelist what `on_destroy` says.
func (r *RCPIPWhitelistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RCPIPWhitelistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ContactID.ValueInt64())

	// A state written without the attribute holds no value; treat it as retain.
	if state.OnDestroy.ValueString() != onDestroyClear {
		resp.Diagnostics.AddWarning(
			"RCP IP Whitelist Removed from Terraform State Only",
			fmt.Sprintf("The whitelist of contact %d is no longer managed by Terraform but is still in force at OpenProvider. "+
				"To switch it off on destroy, set on_destroy = \"clear\" and apply before destroying.", id),
		)
		return
	}

	if err := contacts.SetRCPAllowList(r.client, id, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error Clearing RCP IP Whitelist",
			fmt.Sprintf("Could not clear the whitelist of contact %d: %s", id, err.Error()),
		)
		return
	}
	resp.Diagnostics.AddWarning(
		"RCP IP Whitelist Switched Off",
		fmt.Sprintf("The whitelist of contact %d was cleared: the control panel is open to every address.", id),
	)
}

// ImportState imports the whitelist of a contact by its numeric id.
func (r *RCPIPWhitelistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil || id <= 0 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected the numeric contact id, got %q.", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("contact_id"), id)...)
	// Computed with a default; unknown after an import would plan a change.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("on_destroy"), onDestroyRetain)...)
}
