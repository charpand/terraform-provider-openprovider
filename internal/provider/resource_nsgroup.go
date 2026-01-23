// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
							MarkdownDescription: "The IPv4 address of the nameserver (optional).",
							Optional:            true,
						},
						"ip6": schema.StringAttribute{
							MarkdownDescription: "The IPv6 address of the nameserver (optional).",
							Optional:            true,
						},
					},
				},
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
	var plan NSGroupModel
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
	group, err := nsgroups.Create(r.client, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NS Group",
			fmt.Sprintf("Could not create nameserver group %s: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	// Set ID
	plan.ID = types.StringValue(strconv.Itoa(group.ID))

	// Update plan with response values
	plan.Name = types.StringValue(group.Name)

	// Map nameservers from response
	if len(group.Nameservers) > 0 {
		plan.Nameservers = make([]NSGroupNameserverModel, len(group.Nameservers))
		for i, ns := range group.Nameservers {
			plan.Nameservers[i] = NSGroupNameserverModel{
				Name: types.StringValue(ns.Name),
			}
			if ns.IP != "" {
				plan.Nameservers[i].IP = types.StringValue(ns.IP)
			} else {
				plan.Nameservers[i].IP = types.StringNull()
			}
			if ns.IP6 != "" {
				plan.Nameservers[i].IP6 = types.StringValue(ns.IP6)
			} else {
				plan.Nameservers[i].IP6 = types.StringNull()
			}
		}
	}

	// Save state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *NSGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NSGroupModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID
	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid NS Group ID",
			fmt.Sprintf("Could not parse NS group ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Get NS group
	group, err := nsgroups.Get(r.client, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NS Group",
			fmt.Sprintf("Could not read nameserver group %d: %s", id, err.Error()),
		)
		return
	}

	if group == nil {
		// NS group not found - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Map API response to state
	state.ID = types.StringValue(strconv.Itoa(group.ID))
	state.Name = types.StringValue(group.Name)

	// Map nameservers
	if len(group.Nameservers) > 0 {
		state.Nameservers = make([]NSGroupNameserverModel, len(group.Nameservers))
		for i, ns := range group.Nameservers {
			state.Nameservers[i] = NSGroupNameserverModel{
				Name: types.StringValue(ns.Name),
			}
			if ns.IP != "" {
				state.Nameservers[i].IP = types.StringValue(ns.IP)
			} else {
				state.Nameservers[i].IP = types.StringNull()
			}
			if ns.IP6 != "" {
				state.Nameservers[i].IP6 = types.StringValue(ns.IP6)
			} else {
				state.Nameservers[i].IP6 = types.StringNull()
			}
		}
	} else {
		state.Nameservers = []NSGroupNameserverModel{}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *NSGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NSGroupModel
	var state NSGroupModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID
	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid NS Group ID",
			fmt.Sprintf("Could not parse NS group ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

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
	_, err = nsgroups.Update(r.client, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NS Group",
			fmt.Sprintf("Could not update nameserver group %d: %s", id, err.Error()),
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

// Delete deletes the resource and removes the Terraform state on success.
func (r *NSGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NSGroupModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID
	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid NS Group ID",
			fmt.Sprintf("Could not parse NS group ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Delete the NS group
	err = nsgroups.Delete(r.client, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting NS Group",
			fmt.Sprintf("Could not delete nameserver group %d: %s", id, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *NSGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID can be either numeric ID or name
	importID := req.ID

	// Try to parse as numeric ID first
	if id, err := strconv.Atoi(importID); err == nil {
		// It's a numeric ID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(id))...)
		return
	}

	// Otherwise, treat it as a name and look it up
	group, err := nsgroups.GetByName(r.client, importID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NS Group",
			fmt.Sprintf("Could not find nameserver group with name %s: %s", importID, err.Error()),
		)
		return
	}

	// Set the ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(group.ID))...)
}
