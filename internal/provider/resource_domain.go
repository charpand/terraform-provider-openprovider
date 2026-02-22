// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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

// dnssecKeysAttrTypes defines the attribute types for DNSSEC keys.
// This is used consistently across Create, Read, and Update operations.
var dnssecKeysAttrTypes = map[string]attr.Type{
	"algorithm":  types.Int64Type,
	"flags":      types.Int64Type,
	"protocol":   types.Int64Type,
	"public_key": types.StringType,
}

// DomainResource is the resource implementation.
type DomainResource struct {
	client *client.Client
}

// convertDnssecKeysToAPI converts DNSSEC keys from Terraform state to API format.
func convertDnssecKeysToAPI(ctx context.Context, keysList types.List, diags *diag.Diagnostics) []domains.DnssecKey {
	if keysList.IsNull() || len(keysList.Elements()) == 0 {
		return nil
	}

	var keys []DnssecKeyModel
	diags.Append(keysList.ElementsAs(ctx, &keys, false)...)
	if diags.HasError() {
		return nil
	}

	apiKeys := make([]domains.DnssecKey, 0, len(keys))
	for _, key := range keys {
		apiKeys = append(apiKeys, domains.DnssecKey{
			Alg:      int(key.Algorithm.ValueInt64()),
			Flags:    int(key.Flags.ValueInt64()),
			Protocol: int(key.Protocol.ValueInt64()),
			PubKey:   key.PublicKey.ValueString(),
		})
	}
	return apiKeys
}

// mapDnssecKeysToState converts DNSSEC keys from API format to Terraform state.
func mapDnssecKeysToState(ctx context.Context, keys []domains.DnssecKey, diags *diag.Diagnostics) types.List {
	if len(keys) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: dnssecKeysAttrTypes,
		})
	}

	stateKeys := make([]DnssecKeyModel, 0, len(keys))
	for _, key := range keys {
		stateKeys = append(stateKeys, DnssecKeyModel{
			Algorithm: types.Int64Value(int64(key.Alg)),
			Flags:     types.Int64Value(int64(key.Flags)),
			Protocol:  types.Int64Value(int64(key.Protocol)),
			PublicKey: types.StringValue(key.PubKey),
		})
	}
	listValue, listDiags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: dnssecKeysAttrTypes,
	}, stateKeys)
	diags.Append(listDiags...)
	return listValue
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
		MarkdownDescription: "Manages an OpenProvider domain. Supports both domain registration and domain transfer. To transfer a domain, provide an auth_code.",
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
				MarkdownDescription: "The EPP/Authorization code for domain transfer (also known as transfer code or auth code). This is obtained from the current registrar. When provided, the domain will be transferred instead of registered.",
				Optional:            true,
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
			"period": schema.Int64Attribute{
				MarkdownDescription: "Registration period in years. Only applicable for domain registration (not transfers).",
				Optional:            true,
				Computed:            true,
			},
			"ns_group": schema.StringAttribute{
				MarkdownDescription: "The nameserver group to use for this domain. Use this instead of nameserver blocks.",
				Optional:            true,
			},
			"dnssec_keys": schema.ListNestedAttribute{
				MarkdownDescription: "DNSSEC keys for the domain. Optional.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"algorithm": schema.Int64Attribute{
							MarkdownDescription: "The algorithm number.",
							Required:            true,
						},
						"flags": schema.Int64Attribute{
							MarkdownDescription: "The flags field (typically 257 for KSK or 256 for ZSK).",
							Required:            true,
						},
						"protocol": schema.Int64Attribute{
							MarkdownDescription: "The protocol field (typically 3 for DNSSEC).",
							Required:            true,
						},
						"public_key": schema.StringAttribute{
							MarkdownDescription: "The public key.",
							Required:            true,
						},
					},
				},
			},
			"is_dnssec_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable DNSSEC for the domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "The domain expiration date.",
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
	domainName := plan.Domain.ValueString()
	parts := strings.Split(domainName, ".")
	if len(parts) < 2 {
		resp.Diagnostics.AddError(
			"Invalid Domain Domain",
			fmt.Sprintf("Domain name must include extension (e.g., example.com), got: %s", domainName),
		)
		return
	}

	name := strings.Join(parts[:len(parts)-1], ".")
	extension := parts[len(parts)-1]

	var domain *domains.Domain
	var err error

	// Check if this is a transfer (auth_code provided) or a new registration
	isTransfer := !plan.AuthCode.IsNull() && plan.AuthCode.ValueString() != ""

	if isTransfer {
		// Domain Transfer
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

		domain, err = domains.Transfer(r.client, transferReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Transferring Domain",
				fmt.Sprintf("Could not transfer domain %s: %s", domainName, err.Error()),
			)
			return
		}
	} else {
		// Domain Registration
		createReq := &domains.CreateDomainRequest{}
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

		hasNSGroup := !plan.NSGroup.IsNull() && plan.NSGroup.ValueString() != ""

		// Set ns_group if specified (preferred method)
		if hasNSGroup {
			createReq.NSGroup = plan.NSGroup.ValueString()
		}

		// Set DNSSEC keys if specified
		createReq.DnssecKeys = convertDnssecKeysToAPI(ctx, plan.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		// Set DNSSEC enabled if specified
		if !plan.IsDnssecEnabled.IsNull() {
			enabled := plan.IsDnssecEnabled.ValueBool()
			createReq.IsDnssecEnabled = &enabled
		}

		// Create the domain
		domain, err = domains.Create(r.client, createReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Domain",
				fmt.Sprintf("Could not create domain %s: %s", domainName, err.Error()),
			)
			return
		}
	}

	// Set ID to the domain name
	plan.ID = types.StringValue(domainName)

	// Update plan with computed values from the API response
	plan.Status = types.StringValue(domain.Status)

	// Map contact handles from response
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

	// Map autorenew from response
	if domain.Autorenew == "on" {
		plan.Autorenew = types.BoolValue(true)
	} else {
		plan.Autorenew = types.BoolValue(false)
	}

	// Map ns_group from response
	if domain.NSGroup != "" {
		plan.NSGroup = types.StringValue(domain.NSGroup)
	} else {
		plan.NSGroup = types.StringNull()
	}

	// Map DNSSEC keys from response
	plan.DnssecKeys = mapDnssecKeysToState(ctx, domain.DnssecKeys, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map DNSSEC enabled status from response
	plan.IsDnssecEnabled = types.BoolValue(domain.IsDnssecEnabled)

	// Map expiration date if present
	if domain.ExpirationDate != "" {
		plan.ExpirationDate = types.StringValue(domain.ExpirationDate)
	} else {
		plan.ExpirationDate = types.StringNull()
	}

	// Save state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Domain.ValueString()

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
	state.Domain = types.StringValue(domainName)
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

	// Map ns_group from response
	if domain.NSGroup != "" {
		state.NSGroup = types.StringValue(domain.NSGroup)
	} else {
		state.NSGroup = types.StringNull()
	}

	// Map DNSSEC keys from response
	state.DnssecKeys = mapDnssecKeysToState(ctx, domain.DnssecKeys, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map DNSSEC enabled status from response
	state.IsDnssecEnabled = types.BoolValue(domain.IsDnssecEnabled)

	// Map expiration date if present
	if domain.ExpirationDate != "" {
		state.ExpirationDate = types.StringValue(domain.ExpirationDate)
	} else {
		state.ExpirationDate = types.StringNull()
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

	domainName := state.Domain.ValueString()

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

	// Check if there are any actual user-configured changes.
	// Note: Handle fields (AdminHandle, TechHandle, BillingHandle) only detect changes when
	// the plan value is non-null. This is intentional: clearing a handle (changing from value
	// to null) is not a supported operation in the Openprovider API, so we don't detect it
	// as a change. Users cannot use Terraform to clear handles to null. If a handle in the
	// plan is null, it should match the state value.
	hasChanges := (!plan.AdminHandle.Equal(state.AdminHandle) && !plan.AdminHandle.IsNull()) ||
		(!plan.TechHandle.Equal(state.TechHandle) && !plan.TechHandle.IsNull()) ||
		(!plan.BillingHandle.Equal(state.BillingHandle) && !plan.BillingHandle.IsNull()) ||
		!plan.Autorenew.Equal(state.Autorenew) ||
		!plan.NSGroup.Equal(state.NSGroup) ||
		!plan.DnssecKeys.Equal(state.DnssecKeys) ||
		!plan.IsDnssecEnabled.Equal(state.IsDnssecEnabled)

	// If no changes detected, skip the API call and just refresh state to pick up any
	// server-side changes (e.g., DNSSEC keys or other computed fields updated by the API).
	// This optimization reduces unnecessary API calls when no user-configurable fields change.
	// Unlike other resources that always call Update regardless of field changes, this manual
	// change detection prevents redundant API calls for resources with computed fields that
	// can be updated by the API independently.
	if !hasChanges {
		var readReq resource.ReadRequest
		readReq.State = resp.State
		var readResp resource.ReadResponse
		readResp.State = resp.State
		r.Read(ctx, readReq, &readResp)
		resp.State = readResp.State
		resp.Diagnostics.Append(readResp.Diagnostics...)
		return
	}

	// Create update request with only changed mutable attributes
	// Note: OwnerHandle is not updatable (typically immutable after domain creation)
	updateReq := &domains.UpdateDomainRequest{}

	// Update contact handles if changed
	// Only set values if they are not null in the plan
	if !plan.AdminHandle.Equal(state.AdminHandle) && !plan.AdminHandle.IsNull() {
		updateReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.TechHandle.Equal(state.TechHandle) && !plan.TechHandle.IsNull() {
		updateReq.TechHandle = plan.TechHandle.ValueString()
	}
	if !plan.BillingHandle.Equal(state.BillingHandle) && !plan.BillingHandle.IsNull() {
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

	hasNSGroup := !plan.NSGroup.IsNull() && plan.NSGroup.ValueString() != ""

	// Update ns_group if changed
	if !plan.NSGroup.Equal(state.NSGroup) {
		if hasNSGroup {
			updateReq.NSGroup = plan.NSGroup.ValueString()
		} else {
			// Explicitly clear ns_group if it's being removed
			updateReq.NSGroup = ""
		}
	}

	// Update DNSSEC keys if changed
	if !plan.DnssecKeys.Equal(state.DnssecKeys) {
		updateReq.DnssecKeys = convertDnssecKeysToAPI(ctx, plan.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		// If nil, convert to empty slice to explicitly clear DNSSEC keys
		if updateReq.DnssecKeys == nil {
			updateReq.DnssecKeys = []domains.DnssecKey{}
		}
	}

	// Update DNSSEC enabled if changed
	if !plan.IsDnssecEnabled.Equal(state.IsDnssecEnabled) {
		if !plan.IsDnssecEnabled.IsNull() {
			enabled := plan.IsDnssecEnabled.ValueBool()
			updateReq.IsDnssecEnabled = &enabled
		}
	}

	// Send update
	_, err = domains.Update(r.client, domain.ID, updateReq)
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

// Delete prevents deletion of domains as a safety measure.
func (r *DomainResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Domain Deletion Not Allowed",
		"Domains cannot be deleted through this provider as a safety measure. Domain deletions are irreversible and must be performed manually outside of Terraform. To stop managing a domain, remove it from your Terraform configuration.",
	)
}

// ImportState imports an existing resource into Terraform.
func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the domain name
	domainName := req.ID

	// Set both id and domain to the import ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), domainName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domainName)...)

	// Note: auth_code cannot be retrieved from the API after transfer is initiated
	// Users must provide it in their configuration if the domain was transferred
	resp.Diagnostics.AddWarning(
		"Auth Code Required for Transferred Domains",
		"If this domain was transferred to OpenProvider, the authorization code cannot be retrieved from the API. You must provide the auth_code in your Terraform configuration after import, or the resource will show a diff on the next plan.",
	)
}

// getDomainByName finds a domain by its name using the List API.
// Returns nil if the domain is not found.
func getDomainByName(c *client.Client, domainName string) (*domains.Domain, error) {
	domainList, err := domains.List(c)
	if err != nil {
		return nil, err
	}

	// Search for domain by name
	for _, domain := range domainList {
		fullName := domain.Domain.Name + "." + domain.Domain.Extension
		if fullName == domainName {
			return &domain, nil
		}
	}

	return nil, nil
}
