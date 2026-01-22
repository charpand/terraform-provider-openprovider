// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
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
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

// DomainResource is the resource implementation.
type DomainResource struct {
	client *client.Client
}

// NewDomainResource returns a new instance of the domain resource.
func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

// Metadata returns the resource type name.
func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

// Schema defines the schema for the resource.
func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an OpenProvider domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain identifier (domain name).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name (e.g., example.com).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the domain.",
				Computed:            true,
			},
			"autorenew": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain should auto-renew.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"owner_handle": schema.StringAttribute{
				MarkdownDescription: "The owner contact handle for the domain.",
				Required:            true,
			},
			"admin_handle": schema.StringAttribute{
				MarkdownDescription: "The admin contact handle for the domain.",
				Optional:            true,
				Computed:            true,
			},
			"tech_handle": schema.StringAttribute{
				MarkdownDescription: "The tech contact handle for the domain.",
				Optional:            true,
				Computed:            true,
			},
			"billing_handle": schema.StringAttribute{
				MarkdownDescription: "The billing contact handle for the domain.",
				Optional:            true,
				Computed:            true,
			},
			"period": schema.Int64Attribute{
				MarkdownDescription: "Registration period in years.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse domain name into name and extension
	domainName := plan.Name.ValueString()
	parts := strings.Split(domainName, ".")
	if len(parts) < 2 {
		resp.Diagnostics.AddError(
			"Invalid Domain Name",
			fmt.Sprintf("Domain name must include extension (e.g., example.com), got: %s", domainName),
		)
		return
	}

	name := strings.Join(parts[:len(parts)-1], ".")
	extension := parts[len(parts)-1]

	// Create domain request
	createReq := &client.CreateDomainRequest{}
	createReq.Domain.Name = name
	createReq.Domain.Extension = extension
	
	// Set required owner handle
	createReq.OwnerHandle = plan.OwnerHandle.ValueString()
	
	// Set optional contact handles
	if !plan.AdminHandle.IsNull() {
		createReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.TechHandle.IsNull() {
		createReq.TechHandle = plan.TechHandle.ValueString()
	}
	if !plan.BillingHandle.IsNull() {
		createReq.BillingHandle = plan.BillingHandle.ValueString()
	}
	
	// Set period if specified
	if !plan.Period.IsNull() {
		createReq.Period = int(plan.Period.ValueInt64())
	}
	
	// Set autorenew
	if !plan.Autorenew.IsNull() && plan.Autorenew.ValueBool() {
		createReq.Autorenew = "on"
	} else {
		createReq.Autorenew = "off"
	}

	// Create the domain
	_, err := client.Create(r.client, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Domain",
			fmt.Sprintf("Could not create domain %s: %s", domainName, err.Error()),
		)
		return
	}

	// Set ID to the domain name
	plan.ID = types.StringValue(domainName)

	// Save initial state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Immediately call Read to normalize and hydrate the state
	var readReq resource.ReadRequest
	readReq.State = resp.State
	var readResp resource.ReadResponse
	readResp.State = resp.State
	r.Read(ctx, readReq, &readResp)
	resp.State = readResp.State
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Name.ValueString()

	// Get domain by name (we need to find it via list since API uses ID)
	domain, err := getDomainByName(r.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain",
			fmt.Sprintf("Could not read domain %s: %s", domainName, err.Error()),
		)
		return
	}

	if domain == nil {
		// Domain not found - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Map API response to state
	state.ID = types.StringValue(domainName)
	state.Name = types.StringValue(domainName)
	state.Status = types.StringValue(domain.Status)
	
	// Map contact handles
	state.OwnerHandle = types.StringValue(domain.OwnerHandle)
	state.AdminHandle = types.StringValue(domain.AdminHandle)
	state.TechHandle = types.StringValue(domain.TechHandle)
	state.BillingHandle = types.StringValue(domain.BillingHandle)
	
	// Map autorenew
	if domain.Autorenew == "on" {
		state.Autorenew = types.BoolValue(true)
	} else {
		state.Autorenew = types.BoolValue(false)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainModel
	var state DomainModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Name.ValueString()

	// Get domain to get its ID
	domain, err := getDomainByName(r.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Finding Domain",
			fmt.Sprintf("Could not find domain %s: %s", domainName, err.Error()),
		)
		return
	}

	if domain == nil {
		resp.Diagnostics.AddError(
			"Domain Not Found",
			fmt.Sprintf("Domain %s not found", domainName),
		)
		return
	}

	// Create update request with only changed mutable attributes
	updateReq := &client.UpdateDomainRequest{}

	// Update contact handles if changed
	if !plan.AdminHandle.Equal(state.AdminHandle) {
		updateReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.TechHandle.Equal(state.TechHandle) {
		updateReq.TechHandle = plan.TechHandle.ValueString()
	}
	if !plan.BillingHandle.Equal(state.BillingHandle) {
		updateReq.BillingHandle = plan.BillingHandle.ValueString()
	}

	// Update autorenew if changed
	if !plan.Autorenew.Equal(state.Autorenew) {
		if plan.Autorenew.ValueBool() {
			updateReq.Autorenew = "on"
		} else {
			updateReq.Autorenew = "off"
		}
	}

	// Send update
	_, err = client.Update(r.client, domain.ID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Domain",
			fmt.Sprintf("Could not update domain %s: %s", domainName, err.Error()),
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
func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Name.ValueString()

	// Get domain to get its ID
	domain, err := getDomainByName(r.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Finding Domain",
			fmt.Sprintf("Could not find domain %s: %s", domainName, err.Error()),
		)
		return
	}

	if domain == nil {
		// Domain already doesn't exist
		return
	}

	// Delete the domain
	err = client.Delete(r.client, domain.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain",
			fmt.Sprintf("Could not delete domain %s: %s", domainName, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the domain name
	domainName := req.ID

	// Set both id and name to the import ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), domainName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), domainName)...)
}

// getDomainByName finds a domain by its name using the List API.
// Returns nil if the domain is not found.
func getDomainByName(c *client.Client, domainName string) (*client.Domain, error) {
	domains, err := client.List(c)
	if err != nil {
		return nil, err
	}

	// Search for domain by name
	for _, domain := range domains {
		fullName := domain.Domain.Name + "." + domain.Domain.Extension
		if fullName == domainName {
			return &domain, nil
		}
	}

	return nil, nil
}
