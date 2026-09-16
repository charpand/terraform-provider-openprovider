// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/client/prices"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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

// dnssecEnabledFollowsKeys plans `is_dnssec_enabled` from the prior state only
// while `dnssec_keys` is unchanged. The API turns the flag on or off when the
// keys change, so a plan that carried the prior value over an update to the keys
// would be contradicted by the result.
type dnssecEnabledFollowsKeys struct{}

func (m dnssecEnabledFollowsKeys) Description(ctx context.Context) string {
	return m.MarkdownDescription(ctx)
}

func (m dnssecEnabledFollowsKeys) MarkdownDescription(_ context.Context) string {
	return "Keeps the prior value unless `dnssec_keys` changes."
}

func (m dnssecEnabledFollowsKeys) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	var configKeys, stateKeys types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("dnssec_keys"), &configKeys)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("dnssec_keys"), &stateKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The config is compared, not the plan: a null config keeps the keys the
	// state holds, and the plan's own value can still be unknown here, because
	// attribute modifiers run in no fixed order.
	if configKeys.IsNull() || configKeys.Equal(stateKeys) {
		resp.PlanValue = req.StateValue
	}
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

// guardSpend asks Openprovider what the operation costs and reports the quote
// in minor units, or refuses when it is above `max_cost`. It returns false once
// it has added a diagnostic, so the caller returns without ordering anything.
//
// Every way of not getting a usable quote is a refusal rather than a warning.
// The point of the bound is that a mistake in the configuration costs nothing,
// and a bound that gives way when the quote is unreadable does not hold.
func guardSpend(
	c *client.Client,
	plan *DomainModel,
	name, extension, operation string,
	diags *diag.Diagnostics,
) (int64, bool) {
	currency := "EUR"
	if !plan.Currency.IsNull() && plan.Currency.ValueString() != "" {
		currency = plan.Currency.ValueString()
	}

	period := plan.Period.ValueInt64()
	if plan.Period.IsNull() || plan.Period.IsUnknown() || period < 1 {
		period = 1
	}

	quote, err := prices.Create(c, name, extension, period)
	if err != nil {
		diags.AddError(
			"Could Not Read the Price",
			fmt.Sprintf("Openprovider was not asked to %s %s.%s, because its price could not be read: %s", operation, name, extension, err.Error()),
		)
		return 0, false
	}

	charge := quote.Charge()
	if charge.Price <= 0 {
		diags.AddError(
			"Could Not Read the Price",
			fmt.Sprintf("Openprovider quoted no price for %s.%s, so the max_cost bound cannot be held and nothing was ordered.", name, extension),
		)
		return 0, false
	}

	if charge.Currency != currency {
		diags.AddError(
			"Quote Is In Another Currency",
			fmt.Sprintf(
				"max_cost is stated in %s and Openprovider quoted %s for %s.%s. Nothing was ordered: converting the two here would decide with a rate this provider does not know.",
				currency, charge.Currency, name, extension,
			),
		)
		return 0, false
	}

	// Rounded up, so a bound is never passed by a fraction of a cent.
	cost := int64(math.Ceil(charge.Price * 100))
	if cost > plan.MaxCost.ValueInt64() {
		diags.AddError(
			"Costs More Than max_cost",
			fmt.Sprintf(
				"Openprovider quoted %d %s cents to %s %s.%s, above the max_cost of %d. Nothing was ordered.",
				cost, currency, operation, name, extension, plan.MaxCost.ValueInt64(),
			),
		)
		return 0, false
	}

	return cost, true
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tech_handle": schema.StringAttribute{
				MarkdownDescription: "The tech contact handle for the domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"billing_handle": schema.StringAttribute{
				MarkdownDescription: "The billing contact handle for the domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"max_cost": schema.Int64Attribute{
				MarkdownDescription: "The most, in minor units of `currency` (cents for EUR and USD), that this registration or transfer may cost. The live quote is read before anything is ordered, and the apply fails without spending if the quote is higher. No bound is held when this is unset.",
				Optional:            true,
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "The currency `max_cost` is stated in. Defaults to EUR. A quote that comes back in another currency fails the apply rather than being converted, because a wrong conversion would spend money.",
				Optional:            true,
			},
			"cost": schema.Int64Attribute{
				MarkdownDescription: "What the operation was quoted at, in minor units of `currency`, at the time it ran.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
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
					dnssecEnabledFollowsKeys{},
				},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "The domain expiration date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	// Read the live quote and hold it against `max_cost` before anything is
	// ordered. An apply that would spend more than it was told to must fail
	// having spent nothing, so this runs ahead of both branches below. With no
	// `max_cost` there is no bound to hold, and the API is not asked.
	plan.Cost = types.Int64Null()
	if !plan.MaxCost.IsNull() && !plan.MaxCost.IsUnknown() {
		operation := "create"
		if isTransfer {
			operation = "transfer"
		}

		cost, ok := guardSpend(r.client, &plan, name, extension, operation, &resp.Diagnostics)
		if !ok {
			return
		}
		plan.Cost = types.Int64Value(cost)
	}

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
		r.refreshAfterUpdate(ctx, plan, resp)
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
		// An unknown flag is one the keys decide: the API sets it with them, and
		// the value read from an unknown is false, which would turn DNSSEC off.
		if !plan.IsDnssecEnabled.IsNull() && !plan.IsDnssecEnabled.IsUnknown() {
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

	r.refreshAfterUpdate(ctx, plan, resp)
}

// refreshAfterUpdate reads the domain back into `resp.State` the way `Read`
// reads it, then carries the order fields over from the plan. The read starts
// from the prior state, and the API has no record of `period`, `max_cost` or
// `currency`: they describe the order, not the domain. Without the carry an
// update leaves them at whatever the prior state held -- null after an
// import -- and the framework rejects the result as inconsistent with the
// plan. This holds whether or not the update sent a request: an update with
// nothing to send still ends in this refresh.
func (r *DomainResource) refreshAfterUpdate(ctx context.Context, plan DomainModel, resp *resource.UpdateResponse) {
	var readReq resource.ReadRequest
	readReq.State = resp.State
	var readResp resource.ReadResponse
	readResp.State = resp.State
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A domain the read no longer finds has left the state; there is nothing
	// to carry the order fields into.
	if readResp.State.Raw.IsNull() {
		resp.State = readResp.State
		return
	}

	var final DomainModel
	resp.Diagnostics.Append(readResp.State.Get(ctx, &final)...)
	if resp.Diagnostics.HasError() {
		return
	}
	final.Period = plan.Period
	final.MaxCost = plan.MaxCost
	final.Currency = plan.Currency
	resp.Diagnostics.Append(resp.State.Set(ctx, &final)...)
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
