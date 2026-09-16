// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nameservers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &GlueRecordResource{}
	_ resource.ResourceWithConfigure   = &GlueRecordResource{}
	_ resource.ResourceWithImportState = &GlueRecordResource{}
)

// GlueRecordResource publishes the addresses of a nameserver named inside the
// zone it serves. Without it an in-bailiwick delegation cannot resolve: the
// registry would answer a query for the zone with a nameserver name that only
// the zone itself can resolve.
type GlueRecordResource struct {
	client *client.Client
}

// GlueRecordModel is the Terraform state model for a glue record.
type GlueRecordModel struct {
	ID        types.String `tfsdk:"id"`
	Domain    types.String `tfsdk:"domain"`
	Subdomain types.String `tfsdk:"subdomain"`
	IPs       types.Set    `tfsdk:"ips"`
}

// NewGlueRecordResource returns a new glue record resource.
func NewGlueRecordResource() resource.Resource {
	return &GlueRecordResource{}
}

// Metadata sets the resource type name.
func (r *GlueRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glue_record"
}

// Schema defines the resource schema.
func (r *GlueRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Publishes the addresses of a nameserver named under a domain you hold, so that the domain can be delegated to a nameserver inside itself.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The full host name of the nameserver, which is also the key Openprovider stores it under.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the nameserver is named under (e.g., example.com).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The label of the nameserver under the domain, without the domain itself (e.g., ns1).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ips": schema.SetAttribute{
				MarkdownDescription: "The addresses to publish for the nameserver. IPv4 and IPv6 are told apart by their form, and Openprovider holds at most one of each.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure attaches the API client.
func (r *GlueRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// hostName is the key Openprovider stores the record under.
func (m GlueRecordModel) hostName() string {
	return fmt.Sprintf("%s.%s", m.Subdomain.ValueString(), m.Domain.ValueString())
}

// split sorts the configured addresses into the one IPv4 and the one IPv6
// Openprovider holds per nameserver. A colon is what tells the two apart.
func split(ctx context.Context, set types.Set) (ipv4 string, ipv6 string, err error) {
	var addresses []string
	if diags := set.ElementsAs(ctx, &addresses, false); diags.HasError() {
		return "", "", fmt.Errorf("could not read the address set")
	}

	for _, address := range addresses {
		if strings.Contains(address, ":") {
			if ipv6 != "" {
				return "", "", fmt.Errorf("openprovider holds one IPv6 address per nameserver, got %s and %s", ipv6, address)
			}
			ipv6 = address
			continue
		}

		if ipv4 != "" {
			return "", "", fmt.Errorf("openprovider holds one IPv4 address per nameserver, got %s and %s", ipv4, address)
		}
		ipv4 = address
	}

	// A glue record with no address is not a weaker record, it is an
	// unresolvable delegation, so it is refused rather than published.
	if ipv4 == "" && ipv6 == "" {
		return "", "", fmt.Errorf("a glue record needs at least one address")
	}

	return ipv4, ipv6, nil
}

// addresses turns the record back into the set the configuration states.
func addresses(ns *nameservers.Nameserver) types.Set {
	var elements []string
	if ns.IP != "" {
		elements = append(elements, ns.IP)
	}
	if ns.IP6 != "" {
		elements = append(elements, ns.IP6)
	}

	set, _ := types.SetValueFrom(context.Background(), types.StringType, elements)
	return set
}

// Create publishes the glue record.
func (r *GlueRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GlueRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipv4, ipv6, err := split(ctx, plan.IPs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Addresses", err.Error())
		return
	}

	name := plan.hostName()
	if err := nameservers.Create(r.client, &nameservers.Nameserver{Name: name, IP: ipv4, IP6: ipv6}); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Glue Record",
			fmt.Sprintf("Could not create glue record %s: %s", name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the record from Openprovider.
func (r *GlueRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GlueRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.ID.ValueString()
	ns, err := nameservers.Get(r.client, name)
	if err == nameservers.ErrNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Glue Record",
			fmt.Sprintf("Could not read glue record %s: %s", name, err.Error()),
		)
		return
	}

	state.IPs = addresses(ns)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update republishes the addresses. The host name is the key, so it forces a
// replacement instead and never reaches here.
func (r *GlueRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GlueRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipv4, ipv6, err := split(ctx, plan.IPs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Addresses", err.Error())
		return
	}

	name := plan.hostName()
	if err := nameservers.Update(r.client, name, &nameservers.Nameserver{Name: name, IP: ipv4, IP6: ipv6}); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Glue Record",
			fmt.Sprintf("Could not update glue record %s: %s", name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete withdraws the glue record.
func (r *GlueRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GlueRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.ID.ValueString()
	if err := nameservers.Delete(r.client, name); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Glue Record",
			fmt.Sprintf("Could not delete glue record %s: %s", name, err.Error()),
		)
	}
}

// ImportState takes the full host name and splits the first label off it.
func (r *GlueRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	label, domain, found := strings.Cut(req.ID, ".")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			fmt.Sprintf("Expected a full nameserver host name (e.g. ns1.example.com), got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subdomain"), label)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
}
