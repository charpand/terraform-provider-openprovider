// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	dnslib "github.com/charpand/terraform-provider-openprovider/internal/client/dns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DNSZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &DNSZoneDataSource{}
)

// DNSZoneDataSource is the data source implementation.
type DNSZoneDataSource struct {
	client *client.Client
}

// DNSZoneDataSourceModel describes the data source data model.
type DNSZoneDataSourceModel struct {
	ZoneName         types.String `tfsdk:"zone_name"`
	Name             types.String `tfsdk:"name"`
	Extension        types.String `tfsdk:"extension"`
	Type             types.String `tfsdk:"type"`
	CreationDate     types.String `tfsdk:"creation_date"`
	ModificationDate types.String `tfsdk:"modification_date"`
	ID               types.String `tfsdk:"id"`
}

// NewDNSZoneDataSource returns a new instance of the DNS zone data source.
func NewDNSZoneDataSource() datasource.DataSource {
	return &DNSZoneDataSource{}
}

// Metadata returns the data source type name.
func (d *DNSZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

// Schema defines the schema for the data source.
func (d *DNSZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read information about a DNS zone.",
		Attributes: map[string]schema.Attribute{
			"zone_name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS zone to retrieve (e.g., example.com).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name part of the zone (e.g., 'example' in 'example.com').",
				Computed:            true,
			},
			"extension": schema.StringAttribute{
				MarkdownDescription: "The extension/TLD of the zone (e.g., 'com' in 'example.com').",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of DNS zone (e.g., 'master', 'slave').",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the zone was created.",
				Computed:            true,
			},
			"modification_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the zone was last modified.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The zone identifier.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DNSZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Read is called when the provider must read data source values in order to update state.
func (d *DNSZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DNSZoneDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := config.ZoneName.ValueString()

	zone, err := dnslib.GetZone(d.client, zoneName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DNS zone",
			fmt.Sprintf("Could not read DNS zone: %s", err.Error()),
		)
		return
	}

	// Map response to state
	config.Name = types.StringValue(zone.Name)
	config.Extension = types.StringValue(zone.Extension)
	config.Type = types.StringValue(zone.Type)
	config.CreationDate = types.StringValue(zone.CreationDate)
	config.ModificationDate = types.StringValue(zone.ModificationDate)
	config.ID = types.StringValue(fmt.Sprintf("%s.%s", zone.Name, zone.Extension))

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
