package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

// oneOfValidator accepts a known string only when it is one of `allowed`.
type oneOfValidator struct {
	allowed []string
}

func (v oneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(v.allowed, ", "))
}

func (v oneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	for _, allowed := range v.allowed {
		if value == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Value",
		fmt.Sprintf("%s is not one of: %s.", value, strings.Join(v.allowed, ", ")),
	)
}

// authCodeConflictValidator refuses `auth_code_wo` when `auth_code` is also
// set, because the two give the same code in two ways.
type authCodeConflictValidator struct{}

func (v authCodeConflictValidator) Description(_ context.Context) string {
	return "value must not be set together with auth_code"
}

func (v authCodeConflictValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v authCodeConflictValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		return
	}
	var authCode types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auth_code"), &authCode)...)
	if resp.Diagnostics.HasError() || authCode.IsNull() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Combination",
		"Set auth_code or auth_code_wo, not both.",
	)
}

// registrationPeriodValidator accepts a known `period` only when it is 1 or
// more, because `Create` orders at least one year for a registration and
// keeps the period in state. A transfer does not send `period`, so the check
// does not apply when `auth_code` or `auth_code_wo` is set. An unknown code can still
// become empty, which is a registration, so the check applies to it.
type registrationPeriodValidator struct{}

func (v registrationPeriodValidator) Description(_ context.Context) string {
	return "value must be at least 1 for a registration"
}

func (v registrationPeriodValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v registrationPeriodValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueInt64()
	if value >= 1 {
		return
	}
	for _, attribute := range []string{"auth_code", "auth_code_wo"} {
		var authCode types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(attribute), &authCode)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !authCode.IsNull() && !authCode.IsUnknown() && authCode.ValueString() != "" {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Value",
		fmt.Sprintf("%d is less than 1. A registration orders at least one year.", value),
	)
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
