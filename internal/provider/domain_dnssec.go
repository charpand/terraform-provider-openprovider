package provider

import (
	"context"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dnssecKeysAttrTypes defines the attribute types for DNSSEC keys.
// This is used consistently across Create, Read, and Update operations.
var dnssecKeysAttrTypes = map[string]attr.Type{
	"algorithm":  types.Int64Type,
	"flags":      types.Int64Type,
	"protocol":   types.Int64Type,
	"public_key": types.StringType,
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

// dnssecKeysToSend converts the DNSSEC keys of a plan to the keys of a
// request. A null or unknown list sends no keys. An empty list sends an empty
// list, which removes the keys of the domain.
func dnssecKeysToSend(ctx context.Context, keysList types.List, diags *diag.Diagnostics) *[]domains.DnssecKey {
	if keysList.IsNull() || keysList.IsUnknown() {
		return nil
	}
	keys := convertDnssecKeysToAPI(ctx, keysList, diags)
	if keys == nil {
		keys = []domains.DnssecKey{}
	}
	return &keys
}

// isEmptyDnssecKeys tells if a list of DNSSEC keys is a known empty list.
func isEmptyDnssecKeys(keys types.List) bool {
	return !keys.IsNull() && !keys.IsUnknown() && len(keys.Elements()) == 0
}

// mapDnssecKeysToState converts DNSSEC keys from API format to Terraform state.
// The API gives no keys as no list. When the prior value is an empty list, for
// example from a configuration that sets `dnssec_keys = []`, no keys keep that
// empty list, or the state is different from the configuration.
func mapDnssecKeysToState(ctx context.Context, keys []domains.DnssecKey, prior types.List, diags *diag.Diagnostics) types.List {
	if len(keys) == 0 {
		if isEmptyDnssecKeys(prior) {
			return prior
		}
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
