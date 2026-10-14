// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/contacts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ContactDataSource{}
	_ datasource.DataSourceWithConfigure = &ContactDataSource{}
)

// ContactDataSourceModel is the state of the contact data source.
type ContactDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
	Role             types.String `tfsdk:"role"`
	CompanyName      types.String `tfsdk:"company_name"`
	Locale           types.String `tfsdk:"locale"`
	Comments         types.String `tfsdk:"comments"`
	VAT              types.String `tfsdk:"vat"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	APIAccessEnabled types.Bool   `tfsdk:"api_access_enabled"`
	RCPAllow         types.Set    `tfsdk:"rcp_allow"`
	RCPDeny          types.Set    `tfsdk:"rcp_deny"`
}

// ContactDataSource is the data source implementation.
type ContactDataSource struct {
	client *client.Client
}

// NewContactDataSource returns a new instance of the contact data source.
func NewContactDataSource() datasource.DataSource {
	return &ContactDataSource{}
}

// Metadata returns the data source type name.
func (d *ContactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

// Schema defines the schema for the data source.
func (d *ContactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a contact (a login of the control panel and API) by its numeric id, including its control panel IP lists.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The numeric id of the contact.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The login name of the contact.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the contact.",
				Computed:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "The role of the contact.",
				Computed:            true,
			},
			"company_name":       schema.StringAttribute{MarkdownDescription: "The company name.", Computed: true},
			"locale":             schema.StringAttribute{MarkdownDescription: "The language of the contact.", Computed: true},
			"comments":           schema.StringAttribute{MarkdownDescription: "Custom notes on the contact.", Computed: true},
			"vat":                schema.StringAttribute{MarkdownDescription: "The VAT number.", Computed: true},
			"is_active":          schema.BoolAttribute{MarkdownDescription: "Whether the contact is active.", Computed: true},
			"api_access_enabled": schema.BoolAttribute{MarkdownDescription: "Whether the contact may use the API.", Computed: true},
			"rcp_allow": schema.SetAttribute{
				MarkdownDescription: "The control panel IP whitelist. Empty when no whitelist is set.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"rcp_deny": schema.SetAttribute{
				MarkdownDescription: "The control panel IP blacklist.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ContactDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read retrieves the contact.
func (d *ContactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ContactDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(config.ID.ValueString())
	if err != nil || id <= 0 {
		resp.Diagnostics.AddError("Invalid Contact ID", fmt.Sprintf("Expected a numeric contact id, got %q.", config.ID.ValueString()))
		return
	}

	contact, err := contacts.Get(d.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Contact", fmt.Sprintf("Could not read contact %d: %s", id, err.Error()))
		return
	}
	if contact == nil {
		resp.Diagnostics.AddError("Contact Not Found", fmt.Sprintf("Contact %d not found", id))
		return
	}

	state := ContactDataSourceModel{
		ID:               types.StringValue(strconv.Itoa(id)),
		Username:         optionalString(contact.Username),
		Email:            optionalString(contact.Email),
		Role:             optionalString(contact.Role),
		CompanyName:      optionalString(contact.CompanyName),
		Locale:           optionalString(contact.Locale),
		Comments:         optionalString(contact.Comments),
		VAT:              optionalString(contact.VAT),
		IsActive:         types.BoolValue(contact.IsActive),
		APIAccessEnabled: types.BoolValue(contact.APIAccessEnabled),
		RCPAllow:         allowSet(reportedAllow(contact)),
		RCPDeny:          allowSet(nil),
	}
	if contact.RCPClientIPList != nil {
		state.RCPDeny = allowSet(contact.RCPClientIPList.Deny)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// optionalString maps an empty API value to null, as the other models do.
func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
