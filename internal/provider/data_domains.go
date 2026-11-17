// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainsDataSource{}
)

// DomainsDataSource is the data source implementation.
type DomainsDataSource struct {
	client *client.Client
}

// NewDomainsDataSource returns a new instance of the domains data source.
func NewDomainsDataSource() datasource.DataSource {
	return &DomainsDataSource{}
}

// Metadata returns the data source type name.
func (d *DomainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

// Schema defines the schema for the data source.
func (d *DomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the domains the account holds. " +
			"With `full_name` set, the list holds that domain or is empty, " +
			"where the `openprovider_domain` data source errors on a domain the account does not hold.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The listing identifier: the `full_name` filter, or `all`.",
				Computed:            true,
			},
			"full_name": schema.StringAttribute{
				MarkdownDescription: "Keep only the domain with this full name (e.g., example.com).",
				Optional:            true,
			},
			"domains": schema.ListNestedAttribute{
				MarkdownDescription: "The domains listed.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The domain's numeric identifier at OpenProvider.",
							Computed:            true,
						},
						"domain": schema.StringAttribute{
							MarkdownDescription: "The domain name.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The current status of the domain.",
							Computed:            true,
						},
						"owner_handle": schema.StringAttribute{
							MarkdownDescription: "The owner contact handle for the domain.",
							Computed:            true,
						},
						"ns_group": schema.StringAttribute{
							MarkdownDescription: "The nameserver group the domain is delegated to, if any.",
							Computed:            true,
						},
						"autorenew": schema.BoolAttribute{
							MarkdownDescription: "Whether the domain is set to auto-renew.",
							Computed:            true,
						},
						"expiration_date": schema.StringAttribute{
							MarkdownDescription: "When the registration expires.",
							Computed:            true,
						},
						"can_renew": schema.BoolAttribute{
							MarkdownDescription: "Whether OpenProvider allows a renewal of the domain now (see `openprovider_domain_renewal`).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read lists the domains.
func (d *DomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainsModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullName := config.FullName.ValueString()
	listed, err := domains.ListWith(d.client, domains.ListOptions{FullName: fullName})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Domains",
			fmt.Sprintf("Could not list domains: %s", err.Error()),
		)
		return
	}

	entries := make([]DomainsEntryModel, 0, len(listed))
	for _, domain := range listed {
		name := domain.Domain.Name + "." + domain.Domain.Extension
		// The filter matches on the name; keep only the exact one.
		if fullName != "" && name != fullName {
			continue
		}
		entries = append(entries, DomainsEntryModel{
			ID:             types.Int64Value(int64(domain.ID)),
			Domain:         types.StringValue(name),
			Status:         types.StringValue(domain.Status),
			OwnerHandle:    types.StringValue(domain.OwnerHandle),
			NSGroup:        types.StringValue(domain.NSGroup),
			Autorenew:      types.BoolValue(domain.Autorenew == "on"),
			ExpirationDate: types.StringValue(domain.ExpirationDate),
			CanRenew:       types.BoolValue(domain.CanRenew),
		})
	}

	id := "all"
	if fullName != "" {
		id = fullName
	}
	state := DomainsModel{
		ID:       types.StringValue(id),
		FullName: config.FullName,
		Domains:  entries,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
