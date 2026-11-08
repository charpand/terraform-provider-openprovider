// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &NSGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &NSGroupDataSource{}
)

// NSGroupDataSource is the data source implementation.
type NSGroupDataSource struct {
	client *client.Client
}

// NewNSGroupDataSource returns a new instance of the NS group data source.
func NewNSGroupDataSource() datasource.DataSource {
	return &NSGroupDataSource{}
}

// Metadata returns the data source type name.
func (d *NSGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nsgroup"
}

// Schema defines the schema for the data source.
func (d *NSGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an OpenProvider nameserver group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The nameserver group identifier.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the nameserver group.",
				Required:            true,
			},
			"allow_missing": schema.BoolAttribute{
				MarkdownDescription: "Whether a group that does not exist is a result instead of an error. " +
					"When `true`, such a group has `exists` set to `false` and an empty `nameservers`. " +
					"A configuration can use this to find out if it must adopt the group or create it. Defaults to `false`.",
				Optional: true,
			},
			"exists": schema.BoolAttribute{
				MarkdownDescription: "Whether the account holds the group.",
				Computed:            true,
			},
			"nameservers": schema.ListNestedAttribute{
				MarkdownDescription: "List of nameservers in the group.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The hostname of the nameserver (e.g., ns1.example.com).",
							Computed:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "The IPv4 address of the nameserver (optional).",
							Computed:            true,
						},
						"ip6": schema.StringAttribute{
							MarkdownDescription: "The IPv6 address of the nameserver (optional).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *NSGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read retrieves the nameserver group information.
func (d *NSGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NSGroupModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupName := config.Name.ValueString()

	group, err := nsgroups.Get(d.client, groupName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NS Group",
			fmt.Sprintf("Could not read nameserver group %s: %s", groupName, err.Error()),
		)
		return
	}

	if group == nil && config.AllowMissing.ValueBool() {
		config.ID = config.Name
		config.Exists = types.BoolValue(false)
		config.Nameservers = []NSGroupNameserverModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
		return
	}

	if group == nil {
		resp.Diagnostics.AddError(
			"NS Group Not Found",
			fmt.Sprintf("Nameserver group %s not found", groupName),
		)
		return
	}

	var state NSGroupModel
	state.ID = types.StringValue(group.Name)
	state.Name = types.StringValue(group.Name)
	state.AllowMissing = config.AllowMissing
	state.Exists = types.BoolValue(true)

	if len(group.Nameservers) > 0 {
		state.Nameservers = make([]NSGroupNameserverModel, len(group.Nameservers))
		for i, ns := range group.Nameservers {
			state.Nameservers[i] = NSGroupNameserverModel{
				Name: types.StringValue(ns.Name),
			}
			if ns.IP != "" {
				state.Nameservers[i].IP = types.StringValue(ns.IP)
			} else {
				state.Nameservers[i].IP = types.StringNull()
			}
			if ns.IP6 != "" {
				state.Nameservers[i].IP6 = types.StringValue(ns.IP6)
			} else {
				state.Nameservers[i].IP6 = types.StringNull()
			}
		}
	} else {
		state.Nameservers = []NSGroupNameserverModel{}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
