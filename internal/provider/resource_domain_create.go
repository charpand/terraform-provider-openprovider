package provider

import (
	"context"
	"fmt"
	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/client/prices"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"math"
)

// orderPeriod is the registration period, in years, that a create quotes and
// orders. A plan that states no period orders one year, so the order is always
// for the period that `guardSpend` held against `max_cost`. The schema rejects
// a stated period below one year for a registration only. A transfer does not
// send a period, so it is quoted with `prices.Transfer` and does not use this.
func orderPeriod(plan *DomainModel) int64 {
	if plan.Period.IsNull() || plan.Period.IsUnknown() || plan.Period.ValueInt64() < 1 {
		return 1
	}
	return plan.Period.ValueInt64()
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

	// A transfer has a price of its own, and its order does not send `period`.
	var quote *prices.Quote
	var err error
	if operation == "transfer" {
		quote, err = prices.Transfer(c, name, extension)
	} else {
		quote, err = prices.Create(c, name, extension, orderPeriod(plan))
	}
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

// Create creates the resource and sets the initial Terraform state.
func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateNameservers(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse domain name into name and extension. The split is on the first
	// dot, matching the API's own extension field: "example.co.uk" is name
	// "example", extension "co.uk", not name "example.co", extension "uk".
	domainName := plan.Domain.ValueString()
	name, extension, ok := domains.SplitFullName(domainName)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Domain Domain",
			fmt.Sprintf("Domain name must include extension (e.g., example.com), got: %s", domainName),
		)
		return
	}

	var domain *domains.Domain
	var err error

	// Check if this is a transfer (auth_code provided) or a new registration.
	// The plan never has a write-only value, so `auth_code_wo` comes from the
	// configuration.
	authCode := plan.AuthCode.ValueString()
	if authCode == "" && !req.Config.Raw.IsNull() {
		var authCodeWO types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auth_code_wo"), &authCodeWO)...)
		if resp.Diagnostics.HasError() {
			return
		}
		authCode = authCodeWO.ValueString()
	}
	isTransfer := authCode != ""

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
		// Domain Transfer. `period` does not apply to a transfer, so one left
		// out of the configuration stays null rather than unknown, which the
		// framework refuses after apply.
		if plan.Period.IsUnknown() {
			plan.Period = types.Int64Null()
		}

		transferReq := &domains.TransferDomainRequest{}
		transferReq.Domain.Name = name
		transferReq.Domain.Extension = extension
		transferReq.AuthCode = authCode
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

		// Send the DNSSEC keys with the transfer, so that the registry has
		// them when the transfer completes. An empty list sends an empty
		// list, so that the domain does not keep the keys of the registrar
		// it comes from.
		transferReq.DnssecKeys = dnssecKeysToSend(ctx, plan.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		// An unknown flag is one the keys decide, the same as in `Update`.
		if !plan.IsDnssecEnabled.IsNull() && !plan.IsDnssecEnabled.IsUnknown() {
			enabled := plan.IsDnssecEnabled.ValueBool()
			transferReq.IsDnssecEnabled = &enabled
		}

		transferReq.AdditionalData = convertAdditionalDataToAPI(ctx, plan.AdditionalData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		transferReq.Nameservers = nameserversToAPI(ctx, plan.Nameservers, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if whois := plan.IsPrivateWhoisEnabled; !whois.IsNull() && !whois.IsUnknown() {
			enabled := whois.ValueBool()
			transferReq.IsPrivateWhoisEnabled = &enabled
		}
		if !plan.Comments.IsNull() && !plan.Comments.IsUnknown() {
			transferReq.Comments = plan.Comments.ValueString()
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

		// Order the period that was quoted. A period left out of the
		// configuration plans unknown, and the API has no record of it to
		// read back, so the state holds the period that was ordered.
		period := orderPeriod(&plan)
		createReq.Period = int(period)
		if plan.Period.IsUnknown() {
			plan.Period = types.Int64Value(period)
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

		createReq.AdditionalData = convertAdditionalDataToAPI(ctx, plan.AdditionalData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.Nameservers = nameserversToAPI(ctx, plan.Nameservers, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if whois := plan.IsPrivateWhoisEnabled; !whois.IsNull() && !whois.IsUnknown() {
			enabled := whois.ValueBool()
			createReq.IsPrivateWhoisEnabled = &enabled
		}
		if !plan.Comments.IsNull() && !plan.Comments.IsUnknown() {
			createReq.Comments = plan.Comments.ValueString()
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

	// The create and transfer responses carry the order (`id`, `status`,
	// `activation_date`) and not the domain's contacts, so a handle the
	// request omitted would stay unknown after apply, which Terraform refuses
	// as a provider bug. The domain is read back the way `Read` reads it, and
	// mapped from that. A read-back that fails does not fail the create: the
	// order stands, and a failed create would record nothing of it. The
	// response is mapped instead, with the plan's known values kept, and the
	// next refresh fills the rest.
	readBack := false
	if registered, err := getDomainByName(r.client, domainName); err != nil {
		resp.Diagnostics.AddWarning(
			"Error Reading Domain After Create",
			fmt.Sprintf("Could not read domain %s back after its creation, so its state holds the creation response until the next refresh: %s", domainName, err.Error()),
		)
	} else if registered != nil && registered.Status != domainStatusDeleted {
		// A listing that lags the order can show only a deleted record of a
		// domain that the account had before. Its status and DNSSEC values
		// are not those of the new order.
		domain = registered
		readBack = true
	}

	// Set ID to the domain name
	plan.ID = types.StringValue(domainName)

	// Update plan with computed values from the API response
	plan.Status = types.StringValue(domain.Status)

	// Map contact handles from response
	plan.OwnerHandle = knownString(domain.OwnerHandle, plan.OwnerHandle)
	plan.AdminHandle = knownString(domain.AdminHandle, plan.AdminHandle)
	plan.TechHandle = knownString(domain.TechHandle, plan.TechHandle)
	plan.BillingHandle = knownString(domain.BillingHandle, plan.BillingHandle)

	// Map autorenew and ns_group from response. The order response can omit
	// them, and then the state keeps the planned values that are known.
	if domain.Autorenew != "" || plan.Autorenew.IsUnknown() || plan.Autorenew.IsNull() {
		plan.Autorenew = types.BoolValue(domain.Autorenew == "on")
	}
	if domain.NSGroup != "" {
		plan.NSGroup = types.StringValue(domain.NSGroup)
	} else if plan.NSGroup.IsUnknown() {
		plan.NSGroup = types.StringNull()
	}

	// While a transfer is in flight, the registry does not have the keys
	// that the transfer sent, so the domain reads back without them. The
	// state keeps the planned values that are known, or the apply is
	// inconsistent with the plan. `Read` keeps them until the transfer
	// completes, and then reads them from the registry.
	keepPlannedDnssec := isTransfer && domainTransferIsInFlight(domain.Status)

	// Map DNSSEC keys from response
	if !keepPlannedDnssec || plan.DnssecKeys.IsUnknown() {
		plan.DnssecKeys = mapDnssecKeysToState(ctx, domain.DnssecKeys, plan.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Map DNSSEC enabled status from response
	if !keepPlannedDnssec || plan.IsDnssecEnabled.IsUnknown() {
		plan.IsDnssecEnabled = types.BoolValue(domain.IsDnssecEnabled)
	}

	// The create and transfer requests carry no lock, so a configured
	// `is_locked` that the domain does not have is set with an update. The
	// order response carries no lock, so without a read-back the provider
	// does not know the lock of the domain. It then sends a configured lock,
	// and keeps an unset lock null until the next refresh. A failed update
	// does not fail the create, for the same reason as a failed read-back.
	// The state then holds the configured value, the next refresh reads the
	// value at OpenProvider, and the next apply sets it again.
	if plan.IsLocked.IsUnknown() || plan.IsLocked.IsNull() {
		if readBack {
			plan.IsLocked = types.BoolValue(domain.IsLocked)
		} else {
			plan.IsLocked = types.BoolNull()
		}
	} else if locked := plan.IsLocked.ValueBool(); !readBack || locked != domain.IsLocked {
		if _, err := domains.Update(r.client, domain.ID, &domains.UpdateDomainRequest{IsLocked: &locked}); err != nil {
			resp.Diagnostics.AddWarning(
				"Error Setting Domain Lock After Create",
				fmt.Sprintf("Could not set the lock of domain %s to %t after its creation, so the next apply tries again: %s", domainName, locked, err.Error()),
			)
		}
	}

	// The settings that the configuration did not set take the value of the
	// domain. A value that the configuration set stays as planned, because the
	// order response can omit it, and `Read` shows a difference later.
	if plan.IsPrivateWhoisEnabled.IsUnknown() {
		plan.IsPrivateWhoisEnabled = types.BoolValue(domain.IsPrivateWhoisEnabled)
	}
	if plan.IsPrivateWhoisAllowed.IsUnknown() {
		plan.IsPrivateWhoisAllowed = types.BoolValue(domain.IsPrivateWhoisAllowed)
	}
	if plan.Comments.IsUnknown() {
		plan.Comments = types.StringValue(domain.Comments)
	}

	// Map expiration date if present
	if domain.ExpirationDate != "" {
		plan.ExpirationDate = types.StringValue(domain.ExpirationDate)
	} else {
		plan.ExpirationDate = types.StringNull()
	}

	// Save state. The state never keeps a write-only value.
	plan.AuthCodeWO = types.StringNull()
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
