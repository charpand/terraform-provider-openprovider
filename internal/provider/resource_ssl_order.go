// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/ssl"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &SSLOrderResource{}
	_ resource.ResourceWithConfigure = &SSLOrderResource{}
)

// SSLOrderResource is the resource implementation.
type SSLOrderResource struct {
	client *client.Client
}

// SSLOrderModel describes the resource data model.
type SSLOrderModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	ProductID              types.Int64  `tfsdk:"product_id"`
	CommonName             types.String `tfsdk:"common_name"`
	BrandName              types.String `tfsdk:"brand_name"`
	Status                 types.String `tfsdk:"status"`
	OrderDate              types.String `tfsdk:"order_date"`
	ActiveDate             types.String `tfsdk:"active_date"`
	ExpirationDate         types.String `tfsdk:"expiration_date"`
	Autorenew              types.Bool   `tfsdk:"autorenew"`
	OwnerHandle            types.String `tfsdk:"owner_handle"`
	AdminHandle            types.String `tfsdk:"admin_handle"`
	BillingHandle          types.String `tfsdk:"billing_handle"`
	TechnicalHandle        types.String `tfsdk:"technical_handle"`
	AdditionalDomains      types.List   `tfsdk:"additional_domains"`
	DomainValidationMethod types.String `tfsdk:"domain_validation_method"`
}

// NewSSLOrderResource returns a new instance of the SSL order resource.
func NewSSLOrderResource() resource.Resource {
	return &SSLOrderResource{}
}

// Metadata returns the resource type name.
func (r *SSLOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_order"
}

// Schema defines the schema for the resource.
func (r *SSLOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an SSL/TLS certificate order.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The SSL order identifier.",
				Computed:            true,
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "The SSL product ID to order.",
				Required:            true,
			},
			"common_name": schema.StringAttribute{
				MarkdownDescription: "The common name (CN) for the SSL certificate (primary domain).",
				Required:            true,
			},
			"brand_name": schema.StringAttribute{
				MarkdownDescription: "The brand name of the SSL certificate.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the SSL order.",
				Computed:            true,
			},
			"order_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the order was placed.",
				Computed:            true,
			},
			"active_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the certificate became active.",
				Computed:            true,
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "The date and time when the certificate expires.",
				Computed:            true,
			},
			"autorenew": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic renewal of the SSL certificate.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"owner_handle": schema.StringAttribute{
				MarkdownDescription: "The handle/ID of the certificate owner contact.",
				Optional:            true,
				Computed:            true,
			},
			"admin_handle": schema.StringAttribute{
				MarkdownDescription: "The handle/ID of the administrative contact.",
				Optional:            true,
				Computed:            true,
			},
			"billing_handle": schema.StringAttribute{
				MarkdownDescription: "The handle/ID of the billing contact.",
				Optional:            true,
				Computed:            true,
			},
			"technical_handle": schema.StringAttribute{
				MarkdownDescription: "The handle/ID of the technical contact.",
				Optional:            true,
				Computed:            true,
			},
			"additional_domains": schema.ListAttribute{
				MarkdownDescription: "List of additional domains to include in the SSL certificate (SANs).",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"domain_validation_method": schema.StringAttribute{
				MarkdownDescription: "The method used to validate domain ownership (dns, http, email, etc.).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("dns"),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SSLOrderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *SSLOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SSLOrderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var additionalDomains []string
	if !plan.AdditionalDomains.IsNull() {
		diags = plan.AdditionalDomains.ElementsAs(ctx, &additionalDomains, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createReq := &ssl.CreateSSLOrderRequest{
		ProductID:              int(plan.ProductID.ValueInt64()),
		CommonName:             plan.CommonName.ValueString(),
		AdditionalDomains:      additionalDomains,
		DomainValidationMethod: plan.DomainValidationMethod.ValueString(),
	}

	if !plan.OwnerHandle.IsNull() {
		createReq.OwnerHandle = plan.OwnerHandle.ValueString()
	}
	if !plan.AdminHandle.IsNull() {
		createReq.AdminHandle = plan.AdminHandle.ValueString()
	}
	if !plan.BillingHandle.IsNull() {
		createReq.BillingHandle = plan.BillingHandle.ValueString()
	}
	if !plan.TechnicalHandle.IsNull() {
		createReq.TechnicalHandle = plan.TechnicalHandle.ValueString()
	}
	if plan.Autorenew.ValueBool() {
		createReq.Autorenew = "on"
	} else {
		createReq.Autorenew = "off"
	}

	order, err := ssl.CreateOrder(r.client, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SSL order",
			fmt.Sprintf("Could not create SSL order: %s", err.Error()),
		)
		return
	}

	// Map response to state
	plan.ID = types.Int64Value(int64(order.ID))
	plan.BrandName = types.StringValue(order.BrandName)
	plan.Status = types.StringValue(order.Status)
	plan.OrderDate = types.StringValue(order.OrderDate)
	plan.ActiveDate = types.StringValue(order.ActiveDate)
	plan.ExpirationDate = types.StringValue(order.ExpirationDate)
	plan.Autorenew = types.BoolValue(order.Autorenew == "on")
	plan.OwnerHandle = types.StringValue(order.OwnerHandle)
	plan.AdminHandle = types.StringValue(order.AdminHandle)
	plan.BillingHandle = types.StringValue(order.BillingHandle)
	plan.TechnicalHandle = types.StringValue(order.TechnicalHandle)

	if len(order.AdditionalDomains) > 0 {
		domainsVal, diags := types.ListValueFrom(ctx, types.StringType, order.AdditionalDomains)
		resp.Diagnostics.Append(diags...)
		plan.AdditionalDomains = domainsVal
	} else {
		plan.AdditionalDomains = types.ListNull(types.StringType)
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *SSLOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SSLOrderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orderID := int(state.ID.ValueInt64())

	order, err := ssl.GetOrder(r.client, orderID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SSL order",
			fmt.Sprintf("Could not read SSL order: %s", err.Error()),
		)
		return
	}

	// Update state
	state.BrandName = types.StringValue(order.BrandName)
	state.Status = types.StringValue(order.Status)
	state.OrderDate = types.StringValue(order.OrderDate)
	state.ActiveDate = types.StringValue(order.ActiveDate)
	state.ExpirationDate = types.StringValue(order.ExpirationDate)
	state.Autorenew = types.BoolValue(order.Autorenew == "on")
	state.OwnerHandle = types.StringValue(order.OwnerHandle)
	state.AdminHandle = types.StringValue(order.AdminHandle)
	state.BillingHandle = types.StringValue(order.BillingHandle)
	state.TechnicalHandle = types.StringValue(order.TechnicalHandle)

	if len(order.AdditionalDomains) > 0 {
		domainsVal, diags := types.ListValueFrom(ctx, types.StringType, order.AdditionalDomains)
		resp.Diagnostics.Append(diags...)
		state.AdditionalDomains = domainsVal
	} else {
		state.AdditionalDomains = types.ListNull(types.StringType)
	}

	// Set state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *SSLOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SSLOrderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orderID := int(plan.ID.ValueInt64())

	updateReq := &ssl.UpdateSSLOrderRequest{}
	if plan.Autorenew.ValueBool() {
		updateReq.Autorenew = "on"
	} else {
		updateReq.Autorenew = "off"
	}

	order, err := ssl.UpdateOrder(r.client, orderID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating SSL order",
			fmt.Sprintf("Could not update SSL order: %s", err.Error()),
		)
		return
	}

	// Update state
	plan.Status = types.StringValue(order.Status)
	plan.ExpirationDate = types.StringValue(order.ExpirationDate)
	plan.Autorenew = types.BoolValue(order.Autorenew == "on")

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *SSLOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SSLOrderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orderID := int(state.ID.ValueInt64())

	err := ssl.CancelOrder(r.client, orderID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting SSL order",
			fmt.Sprintf("Could not cancel SSL order: %s", err.Error()),
		)
		return
	}
}
