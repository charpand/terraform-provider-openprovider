// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DomainCheckDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainCheckDataSource{}
)

// DomainCheckDataSource is the data source implementation.
type DomainCheckDataSource struct {
	client *client.Client
}

// NewDomainCheckDataSource returns a new instance of the domain check data source.
func NewDomainCheckDataSource() datasource.DataSource {
	return &DomainCheckDataSource{}
}

// Metadata returns the data source type name.
func (d *DomainCheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_check"
}

// Schema defines the schema for the data source.
func (d *DomainCheckDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks whether a domain is available to register. " +
			"Unlike the `openprovider_domain` data source, this asks the registry, not the account: " +
			"a domain somebody else holds is reported as taken rather than as an error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain identifier (domain name).",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to check (e.g., example.com).",
				Required:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The availability status the registry reports: `free` when the domain can be registered, `active` when somebody holds it.",
				Computed:            true,
			},
			"available": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain can be registered (`status` is `free`).",
				Computed:            true,
			},
			"is_premium": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain is a premium domain, priced above the extension's standard price.",
				Computed:            true,
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "The registry's reason for the status, where it gives one.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DomainCheckDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Read checks the domain's availability.
func (d *DomainCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainCheckModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := config.Domain.ValueString()
	name, extension, ok := domains.SplitFullName(domainName)
	if !ok {
		resp.Diagnostics.AddAttributeError(
			path.Root("domain"),
			"Invalid Domain Name",
			fmt.Sprintf("Domain %q must be a name and an extension, like example.com", domainName),
		)
		return
	}

	results, err := domains.Check(d.client, []domains.CheckDomain{{Name: name, Extension: extension}})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Checking Domain",
			fmt.Sprintf("Could not check domain %s: %s", domainName, err.Error()),
		)
		return
	}
	if len(results) != 1 {
		resp.Diagnostics.AddError(
			"Error Checking Domain",
			fmt.Sprintf("Expected one check result for domain %s, got %d", domainName, len(results)),
		)
		return
	}
	result := results[0]

	state := DomainCheckModel{
		ID:        types.StringValue(domainName),
		Domain:    types.StringValue(domainName),
		Status:    types.StringValue(result.Status),
		Available: types.BoolValue(result.Status == "free"),
		IsPremium: types.BoolValue(result.IsPremium),
		Reason:    types.StringValue(result.Reason),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
