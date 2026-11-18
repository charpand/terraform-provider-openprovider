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
	_ datasource.DataSource              = &DNSZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &DNSZonesDataSource{}
)

// DNSZonesDataSource lists the DNS zones of the account.
type DNSZonesDataSource struct {
	client *client.Client
}

// DNSZonesModel describes the data source data model.
type DNSZonesModel struct {
	ID    types.String       `tfsdk:"id"`
	Zones []DNSZonesZoneItem `tfsdk:"zones"`
}

// DNSZonesZoneItem is one zone in the listing.
type DNSZonesZoneItem struct {
	ZoneName         types.String `tfsdk:"zone_name"`
	Name             types.String `tfsdk:"name"`
	Extension        types.String `tfsdk:"extension"`
	Type             types.String `tfsdk:"type"`
	Provider         types.String `tfsdk:"provider_name"`
	CreationDate     types.String `tfsdk:"creation_date"`
	ModificationDate types.String `tfsdk:"modification_date"`
}

// NewDNSZonesDataSource returns a new instance of the DNS zones data source.
func NewDNSZonesDataSource() datasource.DataSource {
	return &DNSZonesDataSource{}
}

// Metadata returns the data source type name.
func (d *DNSZonesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zones"
}

// Schema defines the schema for the data source.
func (d *DNSZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every DNS zone of the account, so a configuration can find a zone or react to the zones that exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Always `all`.",
				Computed:            true,
			},
			"zones": schema.ListNestedAttribute{
				MarkdownDescription: "The zones of the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"zone_name": schema.StringAttribute{
							MarkdownDescription: "The full name of the zone (e.g., example.com).",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name part of the zone (`example` in `example.com`).",
							Computed:            true,
						},
						"extension": schema.StringAttribute{
							MarkdownDescription: "The extension of the zone (`com` in `example.com`).",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the zone (e.g., `master`).",
							Computed:            true,
						},
						"provider_name": schema.StringAttribute{
							MarkdownDescription: "The DNS provider that hosts the zone.",
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
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DNSZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Read lists the zones.
func (d *DNSZonesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	zones, err := dnslib.ListZones(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DNS Zones", fmt.Sprintf("Could not list DNS zones: %s", err))
		return
	}

	items := make([]DNSZonesZoneItem, 0, len(zones))
	for _, zone := range zones {
		items = append(items, DNSZonesZoneItem{
			ZoneName:         types.StringValue(zone.Name + "." + zone.Extension),
			Name:             types.StringValue(zone.Name),
			Extension:        types.StringValue(zone.Extension),
			Type:             types.StringValue(zone.Type),
			Provider:         types.StringValue(zone.Provider),
			CreationDate:     types.StringValue(zone.CreationDate),
			ModificationDate: types.StringValue(zone.ModificationDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &DNSZonesModel{ID: types.StringValue("all"), Zones: items})...)
}
