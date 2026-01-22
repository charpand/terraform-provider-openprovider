// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DomainDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainDataSource{}
)

// DomainDataSource is the data source implementation.
type DomainDataSource struct {
	client *client.Client
}

// NewDomainDataSource returns a new instance of the domain data source.
func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

// Metadata returns the data source type name.
func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

// Schema defines the schema for the data source.
func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an OpenProvider domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain identifier (domain name).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name to look up (e.g., example.com).",
				Required:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the domain.",
				Computed:            true,
			},
			"autorenew": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain is set to auto-renew.",
				Computed:            true,
			},
			"owner_handle": schema.StringAttribute{
				MarkdownDescription: "The owner contact handle for the domain.",
				Computed:            true,
			},
			"admin_handle": schema.StringAttribute{
				MarkdownDescription: "The admin contact handle for the domain.",
				Computed:            true,
			},
			"tech_handle": schema.StringAttribute{
				MarkdownDescription: "The tech contact handle for the domain.",
				Computed:            true,
			},
			"billing_handle": schema.StringAttribute{
				MarkdownDescription: "The billing contact handle for the domain.",
				Computed:            true,
			},
			"period": schema.Int64Attribute{
				MarkdownDescription: "Registration period in years.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read retrieves the domain information.
func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainName := config.Name.ValueString()

	// Get domain by name
	domain, err := getDomainByName(d.client, domainName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain",
			fmt.Sprintf("Could not read domain %s: %s", domainName, err.Error()),
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

	// Map to state
	var state DomainModel
	state.ID = types.StringValue(domainName)
	state.Name = types.StringValue(domainName)
	state.Status = types.StringValue(domain.Status)
	
	// Map contact handles
	state.OwnerHandle = types.StringValue(domain.OwnerHandle)
	state.AdminHandle = types.StringValue(domain.AdminHandle)
	state.TechHandle = types.StringValue(domain.TechHandle)
	state.BillingHandle = types.StringValue(domain.BillingHandle)
	
	// Map autorenew
	if domain.Autorenew == "on" {
		state.Autorenew = types.BoolValue(true)
	} else {
		state.Autorenew = types.BoolValue(false)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
