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
	_ datasource.DataSource              = &DNSRecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &DNSRecordsDataSource{}
)

// DNSRecordsDataSource lists the records of a DNS zone.
type DNSRecordsDataSource struct {
	client *client.Client
}

// DNSRecordsModel describes the data source data model.
type DNSRecordsModel struct {
	ID       types.String         `tfsdk:"id"`
	ZoneName types.String         `tfsdk:"zone_name"`
	Records  []DNSRecordsListItem `tfsdk:"records"`
}

// DNSRecordsListItem is one record in the listing.
type DNSRecordsListItem struct {
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	Value            types.String `tfsdk:"value"`
	TTL              types.Int64  `tfsdk:"ttl"`
	Priority         types.Int64  `tfsdk:"priority"`
	CreationDate     types.String `tfsdk:"creation_date"`
	ModificationDate types.String `tfsdk:"modification_date"`
}

// NewDNSRecordsDataSource returns a new instance of the DNS records data source.
func NewDNSRecordsDataSource() datasource.DataSource {
	return &DNSRecordsDataSource{}
}

// Metadata returns the data source type name.
func (d *DNSRecordsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

// Schema defines the schema for the data source.
func (d *DNSRecordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every record of a DNS zone, so a configuration can check for a record before it adds one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The zone name.",
				Computed:            true,
			},
			"zone_name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS zone (e.g., example.com).",
				Required:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "The records of the zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the record, as OpenProvider lists it.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The record type (A, AAAA, CNAME, MX, TXT, ...).",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value of the record.",
							Computed:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "The time-to-live in seconds.",
							Computed:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "The priority of an MX or SRV record, 0 for other records.",
							Computed:            true,
						},
						"creation_date": schema.StringAttribute{
							MarkdownDescription: "The date and time when the record was created.",
							Computed:            true,
						},
						"modification_date": schema.StringAttribute{
							MarkdownDescription: "The date and time when the record was last modified.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DNSRecordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read lists the records of the zone.
func (d *DNSRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DNSRecordsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := config.ZoneName.ValueString()
	records, err := dnslib.ListRecords(d.client, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DNS Records", fmt.Sprintf("Could not list the records of zone %s: %s", zoneName, err))
		return
	}

	items := make([]DNSRecordsListItem, 0, len(records))
	for _, record := range records {
		items = append(items, DNSRecordsListItem{
			Name:             types.StringValue(record.Name),
			Type:             types.StringValue(record.Type),
			Value:            types.StringValue(record.Value),
			TTL:              types.Int64Value(int64(record.TTL)),
			Priority:         types.Int64Value(int64(record.Priority)),
			CreationDate:     types.StringValue(record.CreationDate),
			ModificationDate: types.StringValue(record.ModificationDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &DNSRecordsModel{ID: config.ZoneName, ZoneName: config.ZoneName, Records: items})...)
}
