// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The values `on_destroy` takes.
const (
	onDestroyRetain = "retain"
	onDestroyExpire = "expire"
	onDestroyDelete = "delete"
)

// The domain status of a name the account holds outright, and the two the
// API uses for a name still on its way in: `REQ` for a transfer requested,
// `PEN` for one pending. Every other status (a failed operation, a deleted
// or expired domain, and so on) means the account no longer has a transfer
// in flight to provide an auth_code for.
const (
	domainStatusActive    = "ACT"
	domainStatusRequested = "REQ"
	domainStatusPending   = "PEN"
)

// domainStatusDeleted is the status of a domain that the account had and
// that is now deleted, for example after a transfer to a different
// registrar. The account no longer holds the name.
const domainStatusDeleted = "DEL"

// domainStatusFailed is the status of a domain with a last operation that
// failed, for example a transfer that the registry refused.
const domainStatusFailed = "FAI"

// domainTransferIsInFlight reports whether status is one the API uses for a
// domain whose transfer has been requested but has not yet completed -- the
// one case an authorization code is still in play.
func domainTransferIsInFlight(status string) bool {
	return status == domainStatusRequested || status == domainStatusPending
}

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

// Configure adds the provider configured client to the resource.
func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ImportState imports an existing resource into Terraform.
func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the domain name
	domainName := req.ID

	// Set both id and domain to the import ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), domainName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domainName)...)
	// An imported domain is retained on destroy until the configuration says
	// otherwise, the same as one this provider registered.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("on_destroy"), onDestroyRetain)...)

	// The authorization code of a transfer cannot be read back from the API, so
	// the configuration is the only place it can come from. A name the account
	// already holds needs none: its status is active, and a transfer, if there
	// was one, is complete. So ask for the status and speak only for a name the
	// registry has not handed over yet -- an import of an owned domain is then
	// quiet, which is what the operator lane does on every deploy.
	if r.client == nil {
		return
	}
	domain, err := getDomainByName(r.client, domainName)
	switch {
	case err != nil:
		resp.Diagnostics.AddWarning(
			"Domain Status Not Read",
			fmt.Sprintf("The import of %s succeeded, but its status could not be read: %s. If a transfer of this domain is still in progress, do not set auth_code in the configuration: auth_code triggers replacement, so setting it would plan to replace the domain rather than merely track it. Wait for the transfer to finish instead.", domainName, err.Error()),
		)
	case domain != nil && domainTransferIsInFlight(domain.Status):
		resp.Diagnostics.AddWarning(
			"Auth Code Required for Transferred Domains",
			fmt.Sprintf("Domain %s has status %q, so a transfer of it is not complete. The authorization code that started it cannot be read back from the API, and auth_code triggers replacement, so setting it in the configuration now would plan to replace the domain rather than merely track it. Leave auth_code unset until the transfer finishes and the status becomes %q.", domainName, domain.Status, domainStatusActive),
		)
	}
}

// knownString is the API's value for a string attribute, or the plan's when
// the API has none and the plan's is known: a computed attribute must not
// stay unknown after apply, and a configured one must not change.
func knownString(fromAPI string, planned types.String) types.String {
	if fromAPI == "" && !planned.IsUnknown() {
		return planned
	}
	return types.StringValue(fromAPI)
}

// convertAdditionalDataToAPI gives the `additional_data` of an order, or nil
// for a null map, which sends no `additional_data`.
func convertAdditionalDataToAPI(ctx context.Context, data types.Map, diags *diag.Diagnostics) map[string]string {
	if data.IsNull() || data.IsUnknown() {
		return nil
	}
	var out map[string]string
	diags.Append(data.ElementsAs(ctx, &out, false)...)
	return out
}

// additionalDataChanged tells if `additional_data` changed from the prior
// state to the plan. A null map and an empty map are equal, because the order
// sends `additional_data` for neither.
func additionalDataChanged(prior, plan types.Map) bool {
	if plan.IsUnknown() {
		return false
	}
	if len(prior.Elements()) == 0 && len(plan.Elements()) == 0 {
		return false
	}
	return !prior.Equal(plan)
}
