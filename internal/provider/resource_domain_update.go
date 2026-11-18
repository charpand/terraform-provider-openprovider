package provider

import (
	"context"
	"fmt"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

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

	validateNameservers(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := state.Domain.ValueString()

	if additionalDataChanged(state.AdditionalData, plan.AdditionalData) {
		resp.Diagnostics.AddWarning(
			"Additional Data Not Sent",
			fmt.Sprintf("additional_data of %s changed, but only the registration or the transfer sends it. The change makes no change at OpenProvider. The state keeps the configured value.", domainName),
		)
	}

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
	hasChanges := !plan.OwnerHandle.Equal(state.OwnerHandle) ||
		(!plan.AdminHandle.Equal(state.AdminHandle) && !plan.AdminHandle.IsNull()) ||
		(!plan.TechHandle.Equal(state.TechHandle) && !plan.TechHandle.IsNull()) ||
		(!plan.BillingHandle.Equal(state.BillingHandle) && !plan.BillingHandle.IsNull()) ||
		!plan.Autorenew.Equal(state.Autorenew) ||
		!plan.NSGroup.Equal(state.NSGroup) ||
		!plan.DnssecKeys.Equal(state.DnssecKeys) ||
		!plan.IsDnssecEnabled.Equal(state.IsDnssecEnabled) ||
		lockChanges(plan, state) ||
		whoisPrivacyChanges(plan, state) ||
		commentsChange(plan, state) ||
		nameserversChange(plan, state)

	// If no changes detected, skip the API call and just refresh state to pick up any
	// server-side changes (e.g., DNSSEC keys or other computed fields updated by the API).
	// This optimization reduces unnecessary API calls when no user-configurable fields change.
	// Unlike other resources that always call Update regardless of field changes, this manual
	// change detection prevents redundant API calls for resources with computed fields that
	// can be updated by the API independently.
	if !hasChanges {
		r.refreshAfterUpdate(ctx, plan, state, resp)
		return
	}

	// Create update request with only changed mutable attributes
	updateReq := &domains.UpdateDomainRequest{}

	// Update contact handles if changed
	// Only set values if they are not null in the plan
	if !plan.OwnerHandle.Equal(state.OwnerHandle) {
		updateReq.OwnerHandle = plan.OwnerHandle.ValueString()
	}
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

	// Update DNSSEC keys if changed. An empty list removes the keys.
	if !plan.DnssecKeys.Equal(state.DnssecKeys) {
		updateReq.DnssecKeys = dnssecKeysToSend(ctx, plan.DnssecKeys, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
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

	if lockChanges(plan, state) {
		locked := plan.IsLocked.ValueBool()
		updateReq.IsLocked = &locked
	}

	if whoisPrivacyChanges(plan, state) {
		enabled := plan.IsPrivateWhoisEnabled.ValueBool()
		updateReq.IsPrivateWhoisEnabled = &enabled
	}
	if commentsChange(plan, state) {
		comments := plan.Comments.ValueString()
		updateReq.Comments = &comments
	}
	if nameserversChange(plan, state) {
		updateReq.Nameservers = nameserversToAPI(ctx, plan.Nameservers, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
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

	r.refreshAfterUpdate(ctx, plan, state, resp)
}

// lockChanges reports whether the plan sets `is_locked` to a value that the
// state does not hold. An unknown plan value is not a change: it comes from a
// prior state that holds no lock, such as a state from before the attribute
// existed, and the refresh after the update fills it.
func lockChanges(plan, state DomainModel) bool {
	return !plan.IsLocked.IsUnknown() && !plan.IsLocked.IsNull() && !plan.IsLocked.Equal(state.IsLocked)
}

// refreshAfterUpdate reads the domain back into `resp.State` the way `Read`
// reads it, then carries the order fields and `on_destroy` over from the
// plan. The read starts from the prior state -- `UpdateResourceResponse.State`
// is seeded from `PriorState`, not the plan -- and the API has no record of
// `period`, `max_cost`, `currency` or `on_destroy`: none of them describe the
// domain itself. Without the carry, an update leaves them at whatever the
// prior state held -- null for a resource whose state predates the field, an
// import, or (for `on_destroy`) any state written before the attribute
// existed -- and the framework rejects the result as inconsistent with the
// plan, which defaults `on_destroy` to a known, non-null value. This holds
// whether or not the update sent a request: an update with nothing to send
// still ends in this refresh.
//
// A plan value can itself still be unknown here: `period`'s
// `UseStateForUnknown` only resolves against a *non-null* prior state, so a
// domain whose state predates the field (or was imported before `period` was
// ever set) plans it unknown on every apply, not just the first. Writing an
// unknown straight into the applied state hits the same "invalid result
// object" error `UseStateForUnknown` was meant to prevent, so an unknown
// plan value falls back to the prior state's value -- null, if that's what
// it holds -- which is a known value the framework accepts.
func (r *DomainResource) refreshAfterUpdate(ctx context.Context, plan, state DomainModel, resp *resource.UpdateResponse) {
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
	if final.Period.IsUnknown() {
		final.Period = state.Period
	}
	final.MaxCost = plan.MaxCost
	if final.MaxCost.IsUnknown() {
		final.MaxCost = state.MaxCost
	}
	final.Currency = plan.Currency
	if final.Currency.IsUnknown() {
		final.Currency = state.Currency
	}
	final.OnDestroy = plan.OnDestroy
	if final.OnDestroy.IsUnknown() {
		final.OnDestroy = state.OnDestroy
	}
	// The read keeps an empty list of keys only from its prior state. After
	// an update to `dnssec_keys = []`, the domain reads back with no keys,
	// so the empty list comes from the plan.
	if final.DnssecKeys.IsNull() && isEmptyDnssecKeys(plan.DnssecKeys) {
		final.DnssecKeys = plan.DnssecKeys
	}
	// The API does not give back `additional_data`, so the value is the one
	// of the plan.
	final.AdditionalData = plan.AdditionalData
	resp.Diagnostics.Append(resp.State.Set(ctx, &final)...)
}
