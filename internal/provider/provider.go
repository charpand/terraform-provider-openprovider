package provider

import (
	"context"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OpenproviderProvider defines the provider implementation.
type OpenproviderProvider struct {
	// version is set to the provider version on release, forward it to the Service implementation.
	version string
}

// OpenproviderProviderModel describes the provider data model.
type OpenproviderProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
}

func (p *OpenproviderProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openprovider"
	resp.Version = p.version
}

func (p *OpenproviderProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "OpenProvider API endpoint. Defaults to production API.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "OpenProvider username.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "OpenProvider password.",
				Optional:            true,
				Sensitive:           true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "OpenProvider API token.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *OpenproviderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OpenproviderProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validation
	var username, password, token, endpoint string

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}

	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	// Validate authentication methods
	// Requirements: óf token is gezet óf zowel username als password zijn gezet
	if token == "" && (username == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"The provider requires either an API token or both username and password for authentication.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Client initialisatie
	c := client.NewClient(client.Config{
		BaseURL:  endpoint,
		Username: username,
		Password: password,
	})

	// If token is provided, we might need to set it directly on the client.
	// Looking at client.go, Client has a Token field, but NewClient doesn't take it in Config.
	if token != "" {
		c.Token = token
	}

	// Client beschikbaar maken
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OpenproviderProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *OpenproviderProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OpenproviderProvider{
			version: version,
		}
	}
}
