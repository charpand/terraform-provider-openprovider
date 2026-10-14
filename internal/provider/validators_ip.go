package provider

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// validIPOrCIDR reports whether s is an IPv4 or IPv6 address, or a CIDR
// block of either. The API documents the entries as plain strings, so this
// accepts only the notations that are certain to mean one thing.
func validIPOrCIDR(s string) bool {
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr.Zone() == ""
	}
	_, err := netip.ParsePrefix(s)
	return err == nil
}

// ipListValidator accepts a known set of strings only when it is not empty
// and every element is an IPv4 or IPv6 address or a CIDR block.
type ipListValidator struct{}

func (ipListValidator) Description(_ context.Context) string {
	return "must hold at least one entry, each an IPv4 address, an IPv6 address or a CIDR block"
}

func (v ipListValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (ipListValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Empty IP List",
			"An empty list switches the IP whitelist off. To lift the restriction, destroy the resource with on_destroy = \"clear\" instead of configuring an empty list.",
		)
		return
	}
	for _, element := range elements {
		s, ok := stringOf(element)
		if !ok {
			continue // unknown until apply
		}
		if !validIPOrCIDR(s) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid IP Address",
				fmt.Sprintf("%q is not an IPv4 address, an IPv6 address or a CIDR block.", s),
			)
		}
	}
}

func stringOf(v attr.Value) (string, bool) {
	s, ok := v.(basetypes.StringValue)
	if !ok || s.IsNull() || s.IsUnknown() {
		return "", false
	}
	return s.ValueString(), true
}
