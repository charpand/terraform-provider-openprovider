// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	ssllib "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &SSLProductDataSource{}
	_ datasource.DataSourceWithConfigure = &SSLProductDataSource{}
)

// SSLProductDataSource is the data source implementation.
type SSLProductDataSource struct {
	client *client.Client
}

// SSLProductDataSourceModel describes the data source data model.
type SSLProductDataSourceModel struct {
	ProductID         types.Int64  `tfsdk:"product_id"`
	Name              types.String `tfsdk:"name"`
	BrandName         types.String `tfsdk:"brand_name"`
	Category          types.String `tfsdk:"category"`
	Description       types.String `tfsdk:"description"`
	DeliveryTime      types.String `tfsdk:"delivery_time"`
	Encryption        types.String `tfsdk:"encryption"`
	FreeRefundDays    types.Int64  `tfsdk:"free_refund_days"`
	FreeReissueDays   types.Int64  `tfsdk:"free_reissue_days"`
	ID                types.String `tfsdk:"id"`
}

// NewSSLProductDataSource returns a new instance of the SSL product data source.
func NewSSLProductDataSource() datasource.DataSource {
	return &SSLProductDataSource{}
}

// Metadata returns the data source type name.
func (d *SSLProductDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_product"
}

// Schema defines the schema for the data source.
func (d *SSLProductDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read information about an SSL/TLS certificate product.",
		Attributes: map[string]schema.Attribute{
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "The SSL product ID to retrieve.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the SSL product.",
				Computed:            true,
			},
			"brand_name": schema.StringAttribute{
				MarkdownDescription: "The brand name of the SSL certificate (e.g., Comodo, Sectigo).",
				Computed:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "The category of the SSL product (e.g., dv, ov, ev).",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the SSL product.",
				Computed:            true,
			},
			"delivery_time": schema.StringAttribute{
				MarkdownDescription: "The estimated delivery time for the SSL certificate.",
				Computed:            true,
			},
			"encryption": schema.StringAttribute{
				MarkdownDescription: "The encryption strength (e.g., 256-bit).",
				Computed:            true,
			},
			"free_refund_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days for free refund after purchase.",
				Computed:            true,
			},
			"free_reissue_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days for free reissue after purchase.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The product identifier.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *SSLProductDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read is called when the provider must read data source values in order to update state.
func (d *SSLProductDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SSLProductDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productID := int(config.ProductID.ValueInt64())

	product, err := ssllib.GetProduct(d.client, productID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SSL product",
			fmt.Sprintf("Could not read SSL product: %s", err.Error()),
		)
		return
	}

	// Map response to state
	config.Name = types.StringValue(product.Name)
	config.BrandName = types.StringValue(product.BrandName)
	config.Category = types.StringValue(product.Category)
	config.Description = types.StringValue(product.Description)
	config.DeliveryTime = types.StringValue(product.DeliveryTime)
	config.Encryption = types.StringValue(product.Encryption)
	config.FreeRefundDays = types.Int64Value(int64(product.FreeRefundDays))
	config.FreeReissueDays = types.Int64Value(int64(product.FreeReissueDays))
	config.ID = types.StringValue(fmt.Sprintf("%d", product.ID))

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
