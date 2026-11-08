package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/tlds"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TLDDataSource{}

// NewTLDDataSource returns a new instance of the extension data source.
func NewTLDDataSource() datasource.DataSource {
	return &TLDDataSource{}
}

// TLDDataSource reads the contact roles of an extension.
type TLDDataSource struct {
	client *client.Client
}

// TLDModel is the state of the data source.
type TLDModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	OwnerHandleSupported    types.Bool   `tfsdk:"owner_handle_supported"`
	AdminHandleSupported    types.Bool   `tfsdk:"admin_handle_supported"`
	TechHandleSupported     types.Bool   `tfsdk:"tech_handle_supported"`
	BillingHandleSupported  types.Bool   `tfsdk:"billing_handle_supported"`
	ResellerHandleSupported types.Bool   `tfsdk:"reseller_handle_supported"`
}

// Metadata returns the data source type name.
func (d *TLDDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tld"
}

func tldRoleAttribute(role string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: fmt.Sprintf("Whether a domain of the extension takes a %s handle. Null when the API does not report it.", role),
		Computed:            true,
	}
}

// Schema defines the schema for the data source.
func (d *TLDDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads which contact roles a domain of an extension takes. " +
			"An extension that Openprovider does not offer is an error. " +
			"The `openprovider_domain_check` data source tells whether a name is available.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the read (the extension).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The extension, without the dot (e.g., `com`).",
				Required:            true,
			},
			"owner_handle_supported":    tldRoleAttribute("owner"),
			"admin_handle_supported":    tldRoleAttribute("admin"),
			"tech_handle_supported":     tldRoleAttribute("tech"),
			"billing_handle_supported":  tldRoleAttribute("billing"),
			"reseller_handle_supported": tldRoleAttribute("reseller"),
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TLDDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the data source.
func (d *TLDDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TLDModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tld, err := tlds.Get(d.client, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Extension",
			fmt.Sprintf("Could not read extension %s: %s", config.Name.ValueString(), err),
		)
		return
	}

	state := TLDModel{
		ID:                      config.Name,
		Name:                    config.Name,
		OwnerHandleSupported:    types.BoolPointerValue(tld.OwnerHandleSupported),
		AdminHandleSupported:    types.BoolPointerValue(tld.AdminHandleSupported),
		TechHandleSupported:     types.BoolPointerValue(tld.TechHandleSupported),
		BillingHandleSupported:  types.BoolPointerValue(tld.BillingHandleSupported),
		ResellerHandleSupported: types.BoolPointerValue(tld.ResellerHandleSupported),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
