// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
	"github.com/charpand/terraform-provider-openprovider/internal/client/prices"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DomainPriceDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainPriceDataSource{}
)

// DomainPriceDataSource is the data source implementation.
type DomainPriceDataSource struct {
	client *client.Client
}

// DomainPriceDataSourceModel describes the data source data model.
type DomainPriceDataSourceModel struct {
	Domain        types.String  `tfsdk:"domain"`
	Operation     types.String  `tfsdk:"operation"`
	Period        types.Int64   `tfsdk:"period"`
	Price         types.Float64 `tfsdk:"price"`
	Currency      types.String  `tfsdk:"currency"`
	ProductPrice  types.Float64 `tfsdk:"product_price"`
	ResellerPrice types.Float64 `tfsdk:"reseller_price"`
	IsPremium     types.Bool    `tfsdk:"is_premium"`
	ID            types.String  `tfsdk:"id"`
}

// NewDomainPriceDataSource returns a new instance of the domain price data source.
func NewDomainPriceDataSource() datasource.DataSource {
	return &DomainPriceDataSource{}
}

// Metadata returns the data source type name.
func (d *DomainPriceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_price"
}

// Schema defines the schema for the data source.
func (d *DomainPriceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads what an operation on a domain costs, without ordering it, " +
			"so a configuration can use the price, for example in a budget check or an output.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The full domain name, for example `example.com`.",
				Required:            true,
			},
			"operation": schema.StringAttribute{
				MarkdownDescription: "What to price: `create` (registration, default), `transfer`, `renew` or `restore`.",
				Optional:            true,
			},
			"period": schema.Int64Attribute{
				MarkdownDescription: "The period in years. Defaults to 1. A transfer is always priced for one year.",
				Optional:            true,
			},
			"price": schema.Float64Attribute{
				MarkdownDescription: "The amount that is charged to the account: the reseller price where OpenProvider quotes one, otherwise the retail price.",
				Computed:            true,
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "The currency of `price`.",
				Computed:            true,
			},
			"product_price": schema.Float64Attribute{
				MarkdownDescription: "The retail price.",
				Computed:            true,
			},
			"reseller_price": schema.Float64Attribute{
				MarkdownDescription: "The price for the reseller, 0 when OpenProvider quotes none.",
				Computed:            true,
			},
			"is_premium": schema.BoolAttribute{
				MarkdownDescription: "Whether the domain is a premium domain.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name, the operation and the period.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DomainPriceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read is called when the provider must read data source values in order to update state.
func (d *DomainPriceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainPriceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()
	name, extension, ok := domains.SplitFullName(domain)
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("domain"), "Invalid domain name",
			fmt.Sprintf("%q is not a domain name such as example.com.", domain))
		return
	}

	operation := "create"
	if !config.Operation.IsNull() {
		operation = config.Operation.ValueString()
	}
	period := int64(1)
	if !config.Period.IsNull() {
		period = config.Period.ValueInt64()
	}
	switch operation {
	case "create", "transfer", "renew", "restore":
	default:
		resp.Diagnostics.AddAttributeError(path.Root("operation"), "Invalid operation",
			fmt.Sprintf("%q is not one of create, transfer, renew and restore.", operation))
		return
	}
	if period < 1 || period > 10 {
		resp.Diagnostics.AddAttributeError(path.Root("period"), "Invalid period",
			fmt.Sprintf("The period is %d years, it must be between 1 and 10.", period))
		return
	}
	if operation == "transfer" {
		period = 1
	}

	quote, err := prices.Get(d.client, operation, name, extension, period)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the domain price", fmt.Sprintf("Could not read the price of %s: %s", domain, err))
		return
	}

	charge := quote.Charge()
	config.Operation = types.StringValue(operation)
	config.Period = types.Int64Value(period)
	config.Price = types.Float64Value(charge.Price)
	config.Currency = types.StringValue(charge.Currency)
	config.ProductPrice = types.Float64Value(quote.Price.Product.Price)
	config.ResellerPrice = types.Float64Value(quote.Price.Reseller.Price)
	config.IsPremium = types.BoolValue(quote.IsPremium)
	config.ID = types.StringValue(fmt.Sprintf("%s/%s/%d", domain, operation, period))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
