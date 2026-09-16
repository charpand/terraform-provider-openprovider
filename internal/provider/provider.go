// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"os"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OpenproviderProvider defines the provider implementation.
// OpenproviderProvider implements the Terraform provider and holds provider-level configuration.
type OpenproviderProvider struct {
	// version is set to the provider version on release; forwarded to the service implementation.
	version string
}

// OpenproviderProviderModel describes the provider data model.
type OpenproviderProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	BaseURL  types.String `tfsdk:"base_url"`
}

// Metadata sets the provider type name and version.
func (p *OpenproviderProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openprovider"
	resp.Version = p.version
}

// Schema defines the provider schema (configuration) exposed to Terraform.
func (p *OpenproviderProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "OpenProvider username. Falls back to the `OPENPROVIDER_USERNAME` environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "OpenProvider password. Falls back to the `OPENPROVIDER_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Root URL of the OpenProvider API. Falls back to the `OPENPROVIDER_BASE_URL` environment variable, and to the production API when neither is set.",
				Optional:            true,
			},
		},
	}
}

// Configure creates and attaches an API client based on provider configuration.
func (p *OpenproviderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OpenproviderProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validation. The configuration wins where it states a value, and the
	// environment answers otherwise, so a credential can reach the provider
	// without being written into the module or into state.
	username := os.Getenv("OPENPROVIDER_USERNAME")
	password := os.Getenv("OPENPROVIDER_PASSWORD")
	baseURL := os.Getenv("OPENPROVIDER_BASE_URL")

	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}

	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	if !data.BaseURL.IsNull() {
		baseURL = data.BaseURL.ValueString()
	}

	if username == "" || password == "" {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"The provider requires both a username and a password, either in the provider block or as OPENPROVIDER_USERNAME and OPENPROVIDER_PASSWORD.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Client initialization
	c := client.NewClient(client.Config{
		Username: username,
		Password: password,
		BaseURL:  baseURL,
	})

	// Make client available
	resp.DataSourceData = c
	resp.ResourceData = c
}

// Resources returns the provider's resources.
func (p *OpenproviderProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomerResource,
		NewDomainResource,
		NewNSGroupResource,
		NewGlueRecordResource,
		NewDNSRecordResource,
		NewSSLOrderResource,
	}
}

// DataSources returns the provider's data sources.
func (p *OpenproviderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCustomerDataSource,
		NewDomainDataSource,
		NewNSGroupDataSource,
		NewDNSZoneDataSource,
		NewSSLProductDataSource,
	}
}

// New returns a provider factory function that creates an `OpenproviderProvider` with the
// provided version string.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OpenproviderProvider{
			version: version,
		}
	}
}
