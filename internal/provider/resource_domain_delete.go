package provider

import (
	"context"
	"fmt"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// destroyDeletedDomain handles a destroy of a domain that the account shows
// only as deleted. It makes no change at OpenProvider and warns, except while
// a transfer is in progress (see `dropDeletedDomain`): the order can still
// bring the domain into the account, with its autorenew on, so the destroy
// fails.
func destroyDeletedDomain(state DomainModel, diags *diag.Diagnostics) {
	domainName := state.Domain.ValueString()
	if domainTransferIsInFlight(state.Status.ValueString()) {
		diags.AddError(
			"Error Deleting Domain",
			fmt.Sprintf("The transfer of domain %s is in progress (status %q), but the account shows only a deleted record of the domain, so the destroy cannot change the domain. Retry the destroy when the transfer completes. If the transfer stopped, remove the domain from state with `terraform state rm`.", domainName, state.Status.ValueString()),
		)
		return
	}
	diags.AddWarning(
		"Domain Left the Account",
		fmt.Sprintf("The account shows domain %s only as deleted, for example after a transfer to a different registrar, so the destroy makes no change to it.", domainName),
	)
}

// Delete does to the domain what `on_destroy` says. With `retain` the domain
// only leaves the state and stays in the account; with `expire` it stays in
// the account with autorenew off; with `delete` it is deleted at OpenProvider.
// Either way the resource leaves the state once this returns without error.
func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A state written before `on_destroy` existed carries no value for it,
	// and retaining is what the previous behaviour amounted to once the
	// resource was removed from the configuration.
	if state.OnDestroy.IsNull() || state.OnDestroy.ValueString() == onDestroyRetain {
		return
	}

	domainName := state.Domain.ValueString()
	domain, err := getDomainByName(r.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain",
			fmt.Sprintf("Could not look up domain %s before deleting it: %s", domainName, err.Error()),
		)
		return
	}
	if domain == nil {
		// The default listing can leave out a domain that is deleted. A
		// domain that the account shows only as deleted left the account, as
		// does a domain that the listing shows as deleted (below).
		deleted, err := getDeletedDomainByName(r.client, domainName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Domain",
				fmt.Sprintf("Could not look up domain %s in the deleted domains of the account, as the listing does not show it: %s", domainName, err.Error()),
			)
			return
		}
		if deleted != nil {
			destroyDeletedDomain(state, &resp.Diagnostics)
			return
		}
		// The listing lags the registry (see `Read`). For `expire`, a domain
		// that the account holds but the listing omits would leave state with
		// its autorenew on, so the availability check confirms the absence.
		if state.OnDestroy.ValueString() == onDestroyExpire {
			free, err := domainIsFree(r.client, domainName)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Deleting Domain",
					fmt.Sprintf("Could not confirm that domain %s left the account, as the listing does not show it: %s", domainName, err.Error()),
				)
				return
			}
			if !free {
				resp.Diagnostics.AddError(
					"Error Deleting Domain",
					fmt.Sprintf("Domain %s is taken, but the account listing does not show it, so its autorenew cannot be turned off. Retry the destroy. If the domain left the account, remove it from state with `terraform state rm`.", domainName),
				)
				return
			}
		}
		// Gone from the account: nothing left to delete.
		return
	}
	if domain.Status == domainStatusDeleted {
		destroyDeletedDomain(state, &resp.Diagnostics)
		return
	}

	if state.OnDestroy.ValueString() == onDestroyExpire {
		// Read from the account, not from state: a renewal turned on outside
		// Terraform renews the domain just the same.
		if domain.Autorenew == "off" {
			return
		}
		if _, err := domains.Update(r.client, domain.ID, &domains.UpdateDomainRequest{Autorenew: "off"}); err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Domain",
				fmt.Sprintf("Could not turn autorenew off for domain %s: %s", domainName, err.Error()),
			)
		}
		return
	}

	if err := domains.Delete(r.client, domain.ID); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain",
			fmt.Sprintf("Could not delete domain %s: %s", domainName, err.Error()),
		)
		return
	}
}
