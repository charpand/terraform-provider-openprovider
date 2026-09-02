// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &NSGroupResource{}
	_ resource.ResourceWithConfigure   = &NSGroupResource{}
	_ resource.ResourceWithImportState = &NSGroupResource{}
)

// NSGroupResource is the resource implementation.
type NSGroupResource struct {
	client *client.Client
}

// NewNSGroupResource returns a new instance of the NS group resource.
func NewNSGroupResource() resource.Resource {
	return &NSGroupResource{}
}

// Metadata returns the resource type name.
func (r *NSGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nsgroup"
}

// Schema defines the schema for the resource.
func (r *NSGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an OpenProvider nameserver group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The nameserver group identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the nameserver group.",
				Required:            true,
			},
			"nameservers": schema.ListNestedAttribute{
				MarkdownDescription: "List of nameservers in the group.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The hostname of the nameserver (e.g., ns1.example.com).",
							Required:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "The IPv4 address of the nameserver. Optional - will be automatically populated by the API if a valid hostname is provided.",
							Optional:            true,
							Computed:            true,
						},
						"ip6": schema.StringAttribute{
							MarkdownDescription: "The IPv6 address of the nameserver. Optional - will be automatically populated by the API if a valid hostname is provided.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
			"allow_deletion": schema.BoolAttribute{
				MarkdownDescription: "Enable deletion of this nameserver group. When false (default), the group is removed from Terraform state but preserved in OpenProvider. Set to true to permit actual deletion.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *NSGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *NSGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NSGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create NS group request
	createReq := &nsgroups.CreateNSGroupRequest{
		Name:        plan.Name.ValueString(),
		Nameservers: make([]nsgroups.Nameserver, len(plan.Nameservers)),
	}

	for i, ns := range plan.Nameservers {
		createReq.Nameservers[i] = nsgroups.Nameserver{
			Name: ns.Name.ValueString(),
		}
		if !ns.IP.IsNull() {
			createReq.Nameservers[i].IP = ns.IP.ValueString()
		}
		if !ns.IP6.IsNull() {
			createReq.Nameservers[i].IP6 = ns.IP6.ValueString()
		}
	}

	// Create the NS group
	name := plan.Name.ValueString()
	if _, err := nsgroups.Create(r.client, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NS Group",
			fmt.Sprintf("Could not create nameserver group %s: %s", name, err.Error()),
		)
		return
	}

	// Read the group back. The create response carries a success flag and
	// nothing else, so it holds none of the values the API decided. Without
	// this read every computed attribute stays unknown, which the framework
	// rejects as an invalid result object, and the apply fails with the group
	// already made.
	group, err := nsgroups.Get(r.client, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NS Group",
			fmt.Sprintf("Created nameserver group %s but could not read it back: %s", name, err.Error()),
		)
		return
	}
	if group == nil || group.Name == "" {
		resp.Diagnostics.AddError(
			"Error Reading NS Group",
			fmt.Sprintf("Created nameserver group %s but the API does not report it.", name),
		)
		return
	}

	// Set ID to the group name (API uses name as identifier)
	plan.ID = types.StringValue(group.Name)

	// Update plan with response values
	plan.Name = types.StringValue(group.Name)
	plan.Nameservers = nameserverState(group.Nameservers, plan.Nameservers)

	// Save state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// nameserverState maps the nameservers the API reports onto provider state.
//
// `ip` and `ip6` are optional and computed, so each one must hold a known
// value once an apply ends. The API gives an address back only for a name it
// keeps one for, and it keeps none for a name it can resolve itself, so an
// address that came from the configuration is held; anything the API and the
// configuration both leave out becomes null.
func nameserverState(reported []nsgroups.Nameserver, configured []NSGroupNameserverModel) []NSGroupNameserverModel {
	held := make(map[string]NSGroupNameserverModel, len(configured))
	for _, ns := range configured {
		held[ns.Name.ValueString()] = ns
	}

	// An empty report leaves the configured names in place, so that the group
	// keeps the shape the apply asked for rather than emptying itself.
	if len(reported) == 0 {
		reported = make([]nsgroups.Nameserver, len(configured))
		for i, ns := range configured {
			reported[i] = nsgroups.Nameserver{Name: ns.Name.ValueString()}
		}
	}

	state := make([]NSGroupNameserverModel, len(reported))
	for i, ns := range reported {
		state[i] = NSGroupNameserverModel{
			Name: types.StringValue(ns.Name),
			IP:   types.StringNull(),
			IP6:  types.StringNull(),
		}
		from, known := held[ns.Name]
		switch {
		case ns.IP != "":
			state[i].IP = types.StringValue(ns.IP)
		case known && !from.IP.IsNull() && !from.IP.IsUnknown():
			state[i].IP = from.IP
		}
		switch {
		case ns.IP6 != "":
			state[i].IP6 = types.StringValue(ns.IP6)
		case known && !from.IP6.IsNull() && !from.IP6.IsUnknown():
			state[i].IP6 = from.IP6
		}
	}
	return state
}

// Read refreshes the Terraform state with the latest data.
func (r *NSGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NSGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID (which is the group name)
	groupName := state.ID.ValueString()

	// Get NS group
	group, err := nsgroups.Get(r.client, groupName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NS Group",
			fmt.Sprintf("Could not read nameserver group %s: %s", groupName, err.Error()),
		)
		return
	}

	// A group that is gone comes back as an empty body rather than as an
	// error, so an empty name is the signal that it no longer exists.
	if group == nil || group.Name == "" {
		// NS group not found - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Map API response to state
	state.ID = types.StringValue(group.Name)
	state.Name = types.StringValue(group.Name)
	state.Nameservers = nameserverState(group.Nameservers, state.Nameservers)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *NSGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NSGroupResourceModel
	var state NSGroupResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID (which is the group name)
	groupName := state.ID.ValueString()

	// Create update request
	updateReq := &nsgroups.UpdateNSGroupRequest{}

	// Update name if changed
	if !plan.Name.Equal(state.Name) {
		updateReq.Name = plan.Name.ValueString()
	}

	// Update nameservers if changed
	nsChanged := len(plan.Nameservers) != len(state.Nameservers)
	if !nsChanged && len(plan.Nameservers) > 0 {
		for i := range plan.Nameservers {
			if i < len(state.Nameservers) {
				if !plan.Nameservers[i].Name.Equal(state.Nameservers[i].Name) ||
					!plan.Nameservers[i].IP.Equal(state.Nameservers[i].IP) ||
					!plan.Nameservers[i].IP6.Equal(state.Nameservers[i].IP6) {
					nsChanged = true
					break
				}
			}
		}
	}

	if nsChanged {
		updateReq.Nameservers = make([]nsgroups.Nameserver, len(plan.Nameservers))
		for i, ns := range plan.Nameservers {
			updateReq.Nameservers[i] = nsgroups.Nameserver{
				Name: ns.Name.ValueString(),
			}
			if !ns.IP.IsNull() {
				updateReq.Nameservers[i].IP = ns.IP.ValueString()
			}
			if !ns.IP6.IsNull() {
				updateReq.Nameservers[i].IP6 = ns.IP6.ValueString()
			}
		}
	}

	// Send update
	_, err := nsgroups.Update(r.client, groupName, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NS Group",
			fmt.Sprintf("Could not update nameserver group %s: %s", groupName, err.Error()),
		)
		return
	}

	// Call Read to refresh the state
	var readReq resource.ReadRequest
	readReq.State = resp.State
	var readResp resource.ReadResponse
	readResp.State = resp.State
	r.Read(ctx, readReq, &readResp)
	resp.State = readResp.State
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

// Delete removes the NS group.
func (r *NSGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NSGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID (which is the group name)
	groupName := state.ID.ValueString()

	// A group may be shared between domains, so removing it can reach further
	// than the configuration that made it. `allow_deletion` decides, the same
	// way it does for `openprovider_dns_record`.
	if !state.AllowDeletion.ValueBool() {
		// Remove from state only - preserve the group in OpenProvider
		resp.Diagnostics.AddWarning(
			"NS Group Removed from Terraform State Only",
			fmt.Sprintf("NS group %s has been removed from your Terraform state but NOT deleted in OpenProvider. "+
				"The nameserver group still exists and can be reimported. "+
				"To enable deletion, set allow_deletion = true on the resource.",
				groupName),
		)
		return
	}

	// Proceed with deletion since allow_deletion is true. Dropping the group
	// from state alone leaves the name taken, so the next create of the same
	// group fails on the collision and the group the state no longer names can
	// only be found by hand.
	if err := nsgroups.Delete(r.client, groupName); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting NS Group",
			fmt.Sprintf("Could not delete nameserver group %s: %s", groupName, err.Error()),
		)
	}
}

// ImportState imports an existing resource into Terraform.
func (r *NSGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the group name (API uses name as identifier)
	groupName := req.ID

	// Set the ID to the group name
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), groupName)...)

	// `allow_deletion` is computed with a default, so leaving it unknown after
	// an import makes the first plan report a change that is not one.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_deletion"), false)...)
}
