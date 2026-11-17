package provider

import (
	"context"
	"sort"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var nameserverAttrTypes = map[string]attr.Type{
	"name": types.StringType,
	"ip":   types.StringType,
	"ip6":  types.StringType,
}

// nameserversToAPI gives the nameservers of a plan for a request, or nil for a
// null or unknown list, which sends no `name_servers`.
func nameserversToAPI(ctx context.Context, list types.List, diags *diag.Diagnostics) []domains.Nameserver {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var models []NameserverModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil
	}
	out := make([]domains.Nameserver, 0, len(models))
	for _, m := range models {
		out = append(out, domains.Nameserver{
			Name: m.Name.ValueString(),
			IP:   m.IP.ValueString(),
			IP6:  m.IP6.ValueString(),
		})
	}
	return out
}

// mapNameserversToState gives the nameservers of a domain for the state. A
// configuration that sets no `nameservers` does not manage them: the domain can
// have nameservers through its `ns_group`, and these must not show as a
// difference, so a null prior stays null.
func mapNameserversToState(ctx context.Context, servers []domains.Nameserver, prior types.List, diags *diag.Diagnostics) types.List {
	nullList := types.ListNull(types.ObjectType{AttrTypes: nameserverAttrTypes})
	if prior.IsNull() || prior.IsUnknown() {
		return nullList
	}

	sorted := append([]domains.Nameserver(nil), servers...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].SeqNr < sorted[j].SeqNr })

	models := make([]NameserverModel, 0, len(sorted))
	for _, server := range sorted {
		models = append(models, NameserverModel{
			Name: types.StringValue(server.Name),
			IP:   stringOrNull(server.IP),
			IP6:  stringOrNull(server.IP6),
		})
	}
	list, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: nameserverAttrTypes}, models)
	diags.Append(listDiags...)
	return list
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// validateNameservers refuses `nameservers` together with `ns_group`: a domain
// has either a nameserver group or a list of its own.
func validateNameservers(plan DomainModel, diags *diag.Diagnostics) {
	hasList := !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown()
	hasGroup := !plan.NSGroup.IsNull() && !plan.NSGroup.IsUnknown() && plan.NSGroup.ValueString() != ""
	if hasList && hasGroup {
		diags.AddError(
			"Conflicting Nameserver Settings",
			"Set either ns_group or nameservers, not both: a domain has a nameserver group or a list of nameservers of its own.",
		)
	}
}

// whoisPrivacyChanges reports whether the plan sets `is_private_whois_enabled`
// to a value that the state does not hold. An unset value keeps the value of
// the domain.
func whoisPrivacyChanges(plan, state DomainModel) bool {
	return !plan.IsPrivateWhoisEnabled.IsUnknown() && !plan.IsPrivateWhoisEnabled.IsNull() &&
		!plan.IsPrivateWhoisEnabled.Equal(state.IsPrivateWhoisEnabled)
}

// commentsChange reports whether the plan sets `comments` to a value that the
// state does not hold. An empty string removes the comments.
func commentsChange(plan, state DomainModel) bool {
	return !plan.Comments.IsUnknown() && !plan.Comments.IsNull() && !plan.Comments.Equal(state.Comments)
}

// nameserversChange reports whether the plan sets `nameservers` to a list that
// the state does not hold. Removing the list from the configuration changes
// nothing at OpenProvider.
func nameserversChange(plan, state DomainModel) bool {
	return !plan.Nameservers.IsUnknown() && !plan.Nameservers.IsNull() && !plan.Nameservers.Equal(state.Nameservers)
}
