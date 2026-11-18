package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dropDeletedDomain removes a domain that the account shows only as deleted
// from state. While a transfer is in progress, a deleted record can be a
// listing that lags the transfer (after a transfer back of a domain that the
// account had before, the listings can show only the old record until the
// order appears), so the state stays as last read and the next apply does not
// order the domain again.
func dropDeletedDomain(ctx context.Context, state DomainModel, resp *resource.ReadResponse) {
	if domainTransferIsInFlight(state.Status.ValueString()) {
		resp.Diagnostics.AddWarning(
			"Transfer Not Listed",
			fmt.Sprintf("The transfer of domain %s is in progress (status %q), but the account shows only a deleted record of the domain, so its state is kept as last read. If the transfer stopped, remove the domain from state with `terraform state rm`.", state.Domain.ValueString(), state.Status.ValueString()),
		)
		return
	}
	resp.State.RemoveResource(ctx)
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
		// The listing lags the registry: a domain just registered is not in it
		// yet, and a listing under load has come back empty for a held name.
		// Dropping the domain from state on that alone has the next apply buy
		// it again. So the absence is confirmed by the availability check,
		// which asks the registry: a name that is free is gone from the
		// account and leaves the state; a name that is taken stays, as the
		// state last read it, until a listing shows it again.
		free, err := domainIsFree(r.client, domainName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Domain",
				fmt.Sprintf("Could not confirm that domain %s left the account, as the listing does not show it: %s", domainName, err.Error()),
			)
			return
		}
		if free {
			resp.State.RemoveResource(ctx)
			return
		}
		// The default listing can leave out a domain that is deleted. A
		// domain that the account shows only as deleted left the account,
		// and a different holder has the name now.
		deleted, err := getDeletedDomainByName(r.client, domainName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Domain",
				fmt.Sprintf("Could not look up domain %s in the deleted domains of the account, as the listing does not show it: %s", domainName, err.Error()),
			)
			return
		}
		if deleted != nil {
			dropDeletedDomain(ctx, state, resp)
			return
		}
		resp.Diagnostics.AddWarning(
			"Domain Not Listed",
			fmt.Sprintf("The account's listing does not show domain %s, but the registry says it is taken, so its state is kept as last read.", domainName),
		)
		return
	}

	// A deleted domain left the account, for example with a transfer to a
	// different registrar.
	if domain.Status == domainStatusDeleted {
		dropDeletedDomain(ctx, state, resp)
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

	// While a transfer is in flight, the registry does not have the keys
	// that the transfer sent (see `Create`), so the state keeps its values.
	// When the transfer completes, the values come from the registry again,
	// and a difference from the configuration plans an update that sends
	// the keys.
	inFlight := domainTransferIsInFlight(domain.Status)

	// Map DNSSEC keys from response
	if !inFlight || state.DnssecKeys.IsNull() {
		state.DnssecKeys = mapDnssecKeysToState(ctx, domain.DnssecKeys, state.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Map DNSSEC enabled status from response
	if !inFlight || state.IsDnssecEnabled.IsNull() {
		state.IsDnssecEnabled = types.BoolValue(domain.IsDnssecEnabled)
	}

	state.IsLocked = types.BoolValue(domain.IsLocked)
	state.IsPrivateWhoisEnabled = types.BoolValue(domain.IsPrivateWhoisEnabled)
	state.IsPrivateWhoisAllowed = types.BoolValue(domain.IsPrivateWhoisAllowed)
	state.Comments = types.StringValue(domain.Comments)
	state.Nameservers = mapNameserversToState(ctx, domain.Nameservers, state.Nameservers, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map expiration date if present
	if domain.ExpirationDate != "" {
		state.ExpirationDate = types.StringValue(domain.ExpirationDate)
	} else {
		state.ExpirationDate = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
