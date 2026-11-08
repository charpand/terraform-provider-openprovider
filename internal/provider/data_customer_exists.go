package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CustomerExistsDataSource{}

// NewCustomerExistsDataSource returns a new instance of the customer exists data source.
func NewCustomerExistsDataSource() datasource.DataSource {
	return &CustomerExistsDataSource{}
}

// CustomerExistsDataSource reads if the account holds a customer handle.
type CustomerExistsDataSource struct {
	client *client.Client
}

// CustomerExistsModel is the state of the data source. It holds only the
// handle and the answer, so that no data of the customer reaches the state.
type CustomerExistsModel struct {
	ID     types.String `tfsdk:"id"`
	Handle types.String `tfsdk:"handle"`
	Exists types.Bool   `tfsdk:"exists"`
}

// Metadata returns the data source type name.
func (d *CustomerExistsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer_exists"
}

// Schema defines the schema for the data source.
func (d *CustomerExistsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads whether the account holds a customer handle. " +
			"Unlike the `openprovider_customer` data source, this keeps the data of the customer out of the state: " +
			"only the handle and the answer reach it. A deleted customer does not exist. A failed read is an error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the read (the handle).",
				Computed:            true,
			},
			"handle": schema.StringAttribute{
				MarkdownDescription: "The customer handle to look up.",
				Required:            true,
			},
			"exists": schema.BoolAttribute{
				MarkdownDescription: "Whether the account holds the handle.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *CustomerExistsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = apiClient
}

// Read looks up the handle and stores only the answer.
func (d *CustomerExistsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CustomerExistsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := customers.Exists(d.client, config.Handle.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Customer",
			fmt.Sprintf("Could not read customer %s: %s", config.Handle.ValueString(), err),
		)
		return
	}

	config.ID = config.Handle
	config.Exists = types.BoolValue(exists)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
