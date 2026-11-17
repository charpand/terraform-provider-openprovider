// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/ssl"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &SSLOrdersDataSource{}
	_ datasource.DataSourceWithConfigure = &SSLOrdersDataSource{}
)

// SSLOrdersDataSource lists the SSL orders of the account.
type SSLOrdersDataSource struct {
	client *client.Client
}

// SSLOrdersModel describes the data source data model.
type SSLOrdersModel struct {
	ID         types.String         `tfsdk:"id"`
	CommonName types.String         `tfsdk:"common_name"`
	Status     types.List           `tfsdk:"status"`
	Orders     []SSLOrdersEntryItem `tfsdk:"orders"`
}

// SSLOrdersEntryItem is one order in the listing.
type SSLOrdersEntryItem struct {
	ID             types.Int64  `tfsdk:"id"`
	CommonName     types.String `tfsdk:"common_name"`
	ProductID      types.Int64  `tfsdk:"product_id"`
	BrandName      types.String `tfsdk:"brand_name"`
	Status         types.String `tfsdk:"status"`
	OrderDate      types.String `tfsdk:"order_date"`
	ActiveDate     types.String `tfsdk:"active_date"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Autorenew      types.Bool   `tfsdk:"autorenew"`
}

// NewSSLOrdersDataSource returns a new instance of the SSL orders data source.
func NewSSLOrdersDataSource() datasource.DataSource {
	return &SSLOrdersDataSource{}
}

// Metadata returns the data source type name.
func (d *SSLOrdersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_orders"
}

// Schema defines the schema for the data source.
func (d *SSLOrdersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the SSL orders of the account, with their status and expiration date, " +
			"so a configuration can find an existing order or warn about one that expires.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`all`, or the common name when `common_name` is set.",
				Computed:            true,
			},
			"common_name": schema.StringAttribute{
				MarkdownDescription: "Only list the orders with exactly this common name.",
				Optional:            true,
			},
			"status": schema.ListAttribute{
				MarkdownDescription: "Only list the orders with one of these statuses (for example `ACT`).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"orders": schema.ListNestedAttribute{
				MarkdownDescription: "The orders of the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The SSL order identifier, as `openprovider_ssl_order` imports it.",
							Computed:            true,
						},
						"common_name": schema.StringAttribute{
							MarkdownDescription: "The common name of the certificate.",
							Computed:            true,
						},
						"product_id": schema.Int64Attribute{
							MarkdownDescription: "The SSL product of the order.",
							Computed:            true,
						},
						"brand_name": schema.StringAttribute{
							MarkdownDescription: "The brand of the certificate.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The status of the order.",
							Computed:            true,
						},
						"order_date": schema.StringAttribute{
							MarkdownDescription: "When the order was placed.",
							Computed:            true,
						},
						"active_date": schema.StringAttribute{
							MarkdownDescription: "When the certificate became active.",
							Computed:            true,
						},
						"expiration_date": schema.StringAttribute{
							MarkdownDescription: "When the certificate expires.",
							Computed:            true,
						},
						"autorenew": schema.BoolAttribute{
							MarkdownDescription: "Whether the order renews automatically.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *SSLOrdersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read lists the orders.
func (d *SSLOrdersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SSLOrdersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var statuses []string
	if !config.Status.IsNull() {
		resp.Diagnostics.Append(config.Status.ElementsAs(ctx, &statuses, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	wantStatus := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		wantStatus[status] = true
	}
	commonName := config.CommonName.ValueString()

	orders, err := ssl.ListOrders(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing SSL Orders", fmt.Sprintf("Could not list SSL orders: %s", err))
		return
	}

	items := make([]SSLOrdersEntryItem, 0, len(orders))
	for _, order := range orders {
		if commonName != "" && order.CommonName != commonName {
			continue
		}
		if len(wantStatus) > 0 && !wantStatus[order.Status] {
			continue
		}
		items = append(items, SSLOrdersEntryItem{
			ID:             types.Int64Value(int64(order.ID)),
			CommonName:     types.StringValue(order.CommonName),
			ProductID:      types.Int64Value(int64(order.ProductID)),
			BrandName:      types.StringValue(order.BrandName),
			Status:         types.StringValue(order.Status),
			OrderDate:      types.StringValue(order.OrderDate),
			ActiveDate:     types.StringValue(order.ActiveDate),
			ExpirationDate: types.StringValue(order.ExpirationDate),
			Autorenew:      types.BoolValue(order.Autorenew == "on"),
		})
	}

	id := "all"
	if commonName != "" {
		id = commonName
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &SSLOrdersModel{
		ID:         types.StringValue(id),
		CommonName: config.CommonName,
		Status:     config.Status,
		Orders:     items,
	})...)
}
