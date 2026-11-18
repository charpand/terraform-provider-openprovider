package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
			"auth_code_wo": schema.StringAttribute{
				MarkdownDescription: "The auth code for a domain transfer, as a write-only argument. Terraform does not keep the value in the plan or in the state, and it can come from an ephemeral value. The provider uses it only when it creates the domain. To transfer again with a new code, change `auth_code_wo_version`. Conflicts with `auth_code`.",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				Validators: []validator.String{
					authCodeConflictValidator{},
				},
			},
			"auth_code_wo_version": schema.Int64Attribute{
				MarkdownDescription: "A version of `auth_code_wo`. A change of the version replaces the domain, so the provider sends the transfer again with the code that `auth_code_wo` has then.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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
			"on_destroy": schema.StringAttribute{
				MarkdownDescription: "What destroying this resource does to the domain at OpenProvider. `retain` (the default) only forgets the domain in state: it stays in the account, and a later `import` takes it back. `expire` also forgets the domain in state, and turns its autorenew off at OpenProvider: the domain stays in the account until its expiration date at OpenProvider, and then expires. For some extensions, that date is some days before the expiration date at the registry. `delete` deletes the domain at OpenProvider through `DELETE /v1/domains/{id}`; where the registry allows it the domain then goes into OpenProvider's soft quarantine, from where a restore is priced separately. When the account shows the domain as deleted (status `DEL`), for example after a transfer to a different registrar, a destroy makes no change at OpenProvider and gives a warning. A destroy uses the value in state, not the value in the configuration: apply a change of `on_destroy` before a destroy. Use `lifecycle { prevent_destroy = true }` to refuse a destroy altogether.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(onDestroyRetain),
				Validators: []validator.String{
					oneOfValidator{allowed: []string{onDestroyRetain, onDestroyExpire, onDestroyDelete}},
				},
			},
			"owner_handle": schema.StringAttribute{
				MarkdownDescription: "The owner contact handle for the domain. A change of this handle changes the owner of the domain " +
					"at the registrar. The provider does not accept a fee for the change: when the registry requires a fee or a trade, " +
					"the registrar refuses the update and the apply fails.",
				Required: true,
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
				MarkdownDescription: "Registration period in years, at least 1. Defaults to 1 when not set. Only applicable for domain registration (not transfers).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					registrationPeriodValidator{},
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
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
				MarkdownDescription: "DNSSEC keys for the domain. Optional. An empty list removes the keys of the domain; when unset, the domain keeps the keys it has. A transfer sends the keys with the transfer request; while the transfer is in progress (status `REQ` or `PEN`), the state keeps the configured keys.",
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
			"is_locked": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain is locked against a transfer to a different registrar. Set it to `false` before you transfer the domain out, and get the authorization code from OpenProvider. If you do not set it, the provider keeps the value that OpenProvider has. Not all extensions have a lock: for such an extension, do not set `is_locked`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_private_whois_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the owner data of the domain is hidden in the WHOIS (WHOIS privacy). If you do not set it, the provider keeps the value that OpenProvider has. Not every extension allows it: see `is_private_whois_allowed`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_private_whois_allowed": schema.BoolAttribute{
				MarkdownDescription: "Whether OpenProvider allows WHOIS privacy for this domain.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"comments": schema.StringAttribute{
				MarkdownDescription: "Comments on the domain, for your own use. An empty string removes them. If you do not set it, the provider keeps the comments that the domain has.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameservers": schema.ListNestedAttribute{
				MarkdownDescription: "A list of nameservers of your own, in order, instead of a nameserver group. Use either this or `ns_group`. If you do not set it, the provider does not manage the nameservers, and removing the list from the configuration changes nothing at OpenProvider.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The host name of the nameserver.",
							Required:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "The IPv4 address, for a nameserver below the domain itself.",
							Optional:            true,
						},
						"ip6": schema.StringAttribute{
							MarkdownDescription: "The IPv6 address, for a nameserver below the domain itself.",
							Optional:            true,
						},
					},
				},
			},
			"additional_data": schema.MapAttribute{
				MarkdownDescription: "Extension-specific data that the registration or the transfer sends as `additional_data`, for example `{ dk_acceptance = \"1\" }` for the acceptance of the terms of the `.dk` registry. Keys and values go to OpenProvider as given. Only the order sends it: a change after the registration or the transfer makes no change at OpenProvider and gives a warning, and the provider does not read it back.",
				ElementType:         types.StringType,
				Optional:            true,
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
