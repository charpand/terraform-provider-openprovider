// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
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
	_ resource.Resource                = &DomainTransferResource{}
	_ resource.ResourceWithConfigure   = &DomainTransferResource{}
	_ resource.ResourceWithImportState = &DomainTransferResource{}
)

// DomainTransferResource is the resource implementation.
type DomainTransferResource struct {
	client *client.Client
}

// NewDomainTransferResource returns a new instance of the domain transfer resource.
func NewDomainTransferResource() resource.Resource {
	return &DomainTransferResource{}
}

// Metadata returns the resource type name.
func (r *DomainTransferResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_transfer"
}

// Schema defines the schema for the resource.
func (r *DomainTransferResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages domain transfer to OpenProvider. Transfers a domain from another registrar using an authorization code.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain identifier (domain name).",
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
				MarkdownDescription: "The EPP/Authorization code for the domain transfer (also known as transfer code or auth code). This is obtained from the current registrar.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the domain. Common values: REQ (transfer requested), ACT (active/completed).",
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
			"ns_group": schema.StringAttribute{
				MarkdownDescription: "The nameserver group to use for this domain.",
				Optional:            true,
			},
			"import_contacts_from_registry": schema.BoolAttribute{
				MarkdownDescription: "Import contact data from registry and create handles after transfer. When enabled, contact handle parameters can be omitted.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"import_nameservers_from_registry": schema.BoolAttribute{
				MarkdownDescription: "Import nameservers from registry after transfer. When enabled, nameserver parameters can be omitted.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_private_whois_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable WHOIS privacy protection for the domain.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "The domain expiration date.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DomainTransferResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *DomainTransferResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainTransferModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := plan.Domain.ValueString()
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

	transferReq := &domains.TransferDomainRequest{}
	transferReq.Domain.Name = name
	transferReq.Domain.Extension = extension
	transferReq.AuthCode = plan.AuthCode.ValueString()
	transferReq.OwnerHandle = plan.OwnerHandle.ValueString()

	if !plan.AdminHandle.IsNull() {
		transferReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.TechHandle.IsNull() {
		transferReq.TechHandle = plan.TechHandle.ValueString()
	}
	if !plan.BillingHandle.IsNull() {
		transferReq.BillingHandle = plan.BillingHandle.ValueString()
	}

	if !plan.Autorenew.IsNull() && plan.Autorenew.ValueBool() {
		transferReq.Autorenew = "on"
	} else {
		transferReq.Autorenew = "off"
	}

	if !plan.NSGroup.IsNull() && plan.NSGroup.ValueString() != "" {
		transferReq.NSGroup = plan.NSGroup.ValueString()
	}

	if !plan.ImportContactsFromRegistry.IsNull() {
		transferReq.ImportContactsFromRegistry = plan.ImportContactsFromRegistry.ValueBool()
	}
	if !plan.ImportNameserversFromRegistry.IsNull() {
		transferReq.ImportNameserversFromRegistry = plan.ImportNameserversFromRegistry.ValueBool()
	}
	if !plan.IsPrivateWhoisEnabled.IsNull() {
		transferReq.IsPrivateWhoisEnabled = plan.IsPrivateWhoisEnabled.ValueBool()
	}

	domain, err := domains.Transfer(r.client, transferReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Transferring Domain",
			fmt.Sprintf("Could not transfer domain %s: %s", domainName, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(domainName)
	plan.Status = types.StringValue(domain.Status)

	plan.OwnerHandle = types.StringValue(domain.OwnerHandle)
	if domain.AdminHandle != "" {
		plan.AdminHandle = types.StringValue(domain.AdminHandle)
	}
	if domain.TechHandle != "" {
		plan.TechHandle = types.StringValue(domain.TechHandle)
	}
	if domain.BillingHandle != "" {
		plan.BillingHandle = types.StringValue(domain.BillingHandle)
	}

	if domain.Autorenew == "on" {
		plan.Autorenew = types.BoolValue(true)
	} else {
		plan.Autorenew = types.BoolValue(false)
	}

	if domain.NSGroup != "" {
		plan.NSGroup = types.StringValue(domain.NSGroup)
	} else {
		plan.NSGroup = types.StringNull()
	}

	if domain.ExpirationDate != "" {
		plan.ExpirationDate = types.StringValue(domain.ExpirationDate)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DomainTransferResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainTransferModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Domain.ValueString()

	domain, err := getDomainByName(r.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain",
			fmt.Sprintf("Could not read domain %s: %s", domainName, err.Error()),
		)
		return
	}

	if domain == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(domainName)
	state.Domain = types.StringValue(domainName)
	state.Status = types.StringValue(domain.Status)

	state.OwnerHandle = types.StringValue(domain.OwnerHandle)
	state.AdminHandle = types.StringValue(domain.AdminHandle)
	state.TechHandle = types.StringValue(domain.TechHandle)
	state.BillingHandle = types.StringValue(domain.BillingHandle)

	if domain.Autorenew == "on" {
		state.Autorenew = types.BoolValue(true)
	} else {
		state.Autorenew = types.BoolValue(false)
	}

	if domain.NSGroup != "" {
		state.NSGroup = types.StringValue(domain.NSGroup)
	} else {
		state.NSGroup = types.StringNull()
	}

	if domain.ExpirationDate != "" {
		state.ExpirationDate = types.StringValue(domain.ExpirationDate)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DomainTransferResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainTransferModel
	var state DomainTransferModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Domain.ValueString()

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

	updateReq := &domains.UpdateDomainRequest{}

	if !plan.AdminHandle.Equal(state.AdminHandle) && !plan.AdminHandle.IsNull() {
		updateReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.TechHandle.Equal(state.TechHandle) && !plan.TechHandle.IsNull() {
		updateReq.TechHandle = plan.TechHandle.ValueString()
	}
	if !plan.BillingHandle.Equal(state.BillingHandle) && !plan.BillingHandle.IsNull() {
		updateReq.BillingHandle = plan.BillingHandle.ValueString()
	}

	if !plan.Autorenew.Equal(state.Autorenew) {
		if plan.Autorenew.ValueBool() {
			updateReq.Autorenew = "on"
		} else {
			updateReq.Autorenew = "off"
		}
	}

	hasNSGroup := !plan.NSGroup.IsNull() && plan.NSGroup.ValueString() != ""

	if !plan.NSGroup.Equal(state.NSGroup) {
		if hasNSGroup {
			updateReq.NSGroup = plan.NSGroup.ValueString()
		} else {
			updateReq.NSGroup = ""
		}
	}

	_, err = domains.Update(r.client, domain.ID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Domain",
			fmt.Sprintf("Could not update domain %s: %s", domainName, err.Error()),
		)
		return
	}

	var readReq resource.ReadRequest
	readReq.State = resp.State
	var readResp resource.ReadResponse
	readResp.State = resp.State
	r.Read(ctx, readReq, &readResp)
	resp.State = readResp.State
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *DomainTransferResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainTransferModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For domain transfers, we only remove from state
	// The domain remains at Openprovider
}

// ImportState imports an existing resource into Terraform.
func (r *DomainTransferResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainName := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), domainName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domainName)...)
}
