// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/contacts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ContactResource{}
	_ resource.ResourceWithConfigure   = &ContactResource{}
	_ resource.ResourceWithImportState = &ContactResource{}
)

var (
	contactNameTypes = map[string]attr.Type{
		"first_name": types.StringType, "last_name": types.StringType,
		"initials": types.StringType, "prefix": types.StringType,
	}
	contactPhoneTypes = map[string]attr.Type{
		"country_code": types.StringType, "area_code": types.StringType, "number": types.StringType,
	}
	contactAddressTypes = map[string]attr.Type{
		"street": types.StringType, "number": types.StringType, "suffix": types.StringType,
		"zipcode": types.StringType, "city": types.StringType, "state": types.StringType,
		"country": types.StringType,
	}
)

// ContactResourceModel is the Terraform state of a contact.
type ContactResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	Email            types.String `tfsdk:"email"`
	Role             types.String `tfsdk:"role"`
	CompanyName      types.String `tfsdk:"company_name"`
	Locale           types.String `tfsdk:"locale"`
	Comments         types.String `tfsdk:"comments"`
	VAT              types.String `tfsdk:"vat"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	APIAccessEnabled types.Bool   `tfsdk:"api_access_enabled"`
	Name             types.Object `tfsdk:"name"`
	Phone            types.Object `tfsdk:"phone"`
	Address          types.Object `tfsdk:"address"`
	OnDestroy        types.String `tfsdk:"on_destroy"`
}

// ContactResource manages a contact.
type ContactResource struct {
	client *client.Client
}

// NewContactResource returns a new instance of the contact resource.
func NewContactResource() resource.Resource {
	return &ContactResource{}
}

// Metadata returns the resource type name.
func (r *ContactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func computedString(description string, required bool) schema.StringAttribute {
	a := schema.StringAttribute{
		MarkdownDescription: description,
		Optional:            !required,
		Computed:            !required,
		Required:            required,
	}
	if !required {
		a.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	}
	return a
}

func nestedStrings(descriptions map[string]string) map[string]schema.Attribute {
	out := make(map[string]schema.Attribute, len(descriptions))
	for name, description := range descriptions {
		out[name] = computedString(description, false)
	}
	return out
}

// Schema defines the schema for the resource.
func (r *ContactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	objectPlan := []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a contact: a login of the OpenProvider control panel (RCP) and API. " +
			"A contact is a different object from a customer (`openprovider_customer`) and has a numeric id.\n\n" +
			"Every optional attribute is also computed: an attribute that is left out of the configuration is not managed, " +
			"and keeps the value the contact has at OpenProvider. This lets you import an existing contact and manage only part of it. " +
			"The contact's IP lists are not managed here; use `openprovider_rcp_ip_whitelist` for the control panel whitelist.\n\n" +
			"~> **Warning:** do not manage the contact Terraform itself logs in with unless you are sure of the change. " +
			"Setting `is_active` or `api_access_enabled` to `false`, or changing `email` or `role`, on that contact can lock Terraform out.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The numeric id of the contact. Use `tonumber(openprovider_contact.example.id)` where a number is expected.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The login name of the contact. It cannot be changed; changing it replaces the contact.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of a new contact. It is write-only: it is sent when the contact is created, and is never stored in the plan or state (this needs Terraform 1.11 or later). " +
					"The API has no way to change it afterwards through this resource, so changing it has no effect. Required to create a contact, ignored for an imported one.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
			},
			"email":        computedString("The email address of the contact.", false),
			"role":         computedString("The role of the contact.", false),
			"company_name": computedString("The company name.", false),
			"locale":       computedString("The language of the contact, for example `en_GB`.", false),
			"comments":     computedString("Custom notes on the contact.", false),
			"vat":          computedString("The VAT number.", false),
			"is_active": schema.BoolAttribute{
				MarkdownDescription: "Whether the contact is active.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"api_access_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the contact may use the API.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"name": schema.SingleNestedAttribute{
				MarkdownDescription: "The name of the contact.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       objectPlan,
				Attributes: nestedStrings(map[string]string{
					"first_name": "The first name.",
					"last_name":  "The last name.",
					"initials":   "The initials.",
					"prefix":     "The name prefix, for example `van der`.",
				}),
			},
			"phone": schema.SingleNestedAttribute{
				MarkdownDescription: "The phone number of the contact.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       objectPlan,
				Attributes: nestedStrings(map[string]string{
					"country_code": "The country calling code, for example `+31`.",
					"area_code":    "The area code.",
					"number":       "The subscriber number.",
				}),
			},
			"address": schema.SingleNestedAttribute{
				MarkdownDescription: "The address of the contact.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       objectPlan,
				Attributes: nestedStrings(map[string]string{
					"street":  "The street.",
					"number":  "The house number.",
					"suffix":  "The house number suffix.",
					"zipcode": "The postal code.",
					"city":    "The city.",
					"state":   "The state or province.",
					"country": "The two-letter country code.",
				}),
			},
			"on_destroy": schema.StringAttribute{
				MarkdownDescription: "What destroying this resource does at OpenProvider. `retain` (the default) only forgets the contact in state: it stays in the account, and a later `import` takes it back. `delete` deletes the contact through `DELETE /v1beta/contacts/{id}`. Use `lifecycle { prevent_destroy = true }` to refuse a destroy altogether.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(onDestroyRetain),
				Validators: []validator.String{
					oneOfValidator{allowed: []string{onDestroyRetain, onDestroyDelete}},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ContactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// known returns the value of s when the configuration or plan sets it.
func known(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

func knownBool(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// objectStrings returns the strings an object sets, or nil for an object that
// is null, unknown or holds nothing.
func objectStrings(obj types.Object) map[string]string {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	out := map[string]string{}
	for k, v := range obj.Attributes() {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() && s.ValueString() != "" {
			out[k] = s.ValueString()
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// request builds the body of a create or an update from the plan; anything
// the plan leaves unset is left out of it.
func contactRequest(plan *ContactResourceModel) *contacts.Request {
	req := &contacts.Request{
		Email:            known(plan.Email),
		Role:             known(plan.Role),
		CompanyName:      known(plan.CompanyName),
		Locale:           known(plan.Locale),
		Comments:         known(plan.Comments),
		VAT:              known(plan.VAT),
		IsActive:         knownBool(plan.IsActive),
		APIAccessEnabled: knownBool(plan.APIAccessEnabled),
	}
	if n := objectStrings(plan.Name); n != nil {
		req.Name = &contacts.Name{FirstName: n["first_name"], LastName: n["last_name"], Initials: n["initials"], Prefix: n["prefix"]}
	}
	if p := objectStrings(plan.Phone); p != nil {
		req.Phone = &contacts.Phone{CountryCode: p["country_code"], AreaCode: p["area_code"], Number: p["number"]}
	}
	if a := objectStrings(plan.Address); a != nil {
		req.Address = &contacts.Address{
			Street: a["street"], Number: a["number"], Suffix: a["suffix"], Zipcode: a["zipcode"],
			City: a["city"], State: a["state"], Country: a["country"],
		}
	}
	return req
}

// pick returns what the API reports, or the value the plan or state already
// holds where the API leaves a field out, or null.
func pick(api string, prior types.String) types.String {
	if api != "" {
		return types.StringValue(api)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringNull()
}

func pickObject(attrTypes map[string]attr.Type, api map[string]string, prior types.Object) types.Object {
	var priorAttrs map[string]attr.Value
	if !prior.IsNull() && !prior.IsUnknown() {
		priorAttrs = prior.Attributes()
	}
	values := make(map[string]attr.Value, len(attrTypes))
	empty := true
	for k := range attrTypes {
		p, _ := priorAttrs[k].(types.String)
		v := pick(api[k], p)
		values[k] = v
		if !v.IsNull() {
			empty = false
		}
	}
	if empty {
		return types.ObjectNull(attrTypes)
	}
	return types.ObjectValueMust(attrTypes, values)
}

// fill sets the model from what the API reports. Values the API leaves out
// keep what the model held.
func (m *ContactResourceModel) fill(c *contacts.Contact) {
	m.ID = types.StringValue(strconv.Itoa(c.ID))
	m.Username = pick(c.Username, m.Username)
	m.Email = pick(c.Email, m.Email)
	m.Role = pick(c.Role, m.Role)
	m.CompanyName = pick(c.CompanyName, m.CompanyName)
	m.Locale = pick(c.Locale, m.Locale)
	m.Comments = pick(c.Comments, m.Comments)
	m.VAT = pick(c.VAT, m.VAT)
	m.IsActive = types.BoolValue(c.IsActive)
	m.APIAccessEnabled = types.BoolValue(c.APIAccessEnabled)
	m.Name = pickObject(contactNameTypes, map[string]string{
		"first_name": c.Name.FirstName, "last_name": c.Name.LastName,
		"initials": c.Name.Initials, "prefix": c.Name.Prefix,
	}, m.Name)
	m.Phone = pickObject(contactPhoneTypes, map[string]string{
		"country_code": c.Phone.CountryCode, "area_code": c.Phone.AreaCode, "number": c.Phone.Number,
	}, m.Phone)
	m.Address = pickObject(contactAddressTypes, map[string]string{
		"street": c.Address.Street, "number": c.Address.Number, "suffix": c.Address.Suffix,
		"zipcode": c.Address.Zipcode, "city": c.Address.City, "state": c.Address.State,
		"country": c.Address.Country,
	}, m.Address)
	// Write-only: never held in state.
	m.Password = types.StringNull()
}

// readBack reads the contact and fills the model from it.
func (r *ContactResource) readBack(m *ContactResourceModel, id int, action string) diag.Diagnostics {
	var diags diag.Diagnostics
	contact, err := contacts.Get(r.client, id)
	if err != nil {
		diags.AddError("Error Reading Contact", fmt.Sprintf("%s contact %d but could not read it back: %s", action, id, err.Error()))
		return diags
	}
	if contact == nil {
		diags.AddError("Error Reading Contact", fmt.Sprintf("%s contact %d but the API does not report it.", action, id))
		return diags
	}
	if contact.ID == 0 {
		contact.ID = id
	}
	m.fill(contact)
	return diags
}

// Create creates the contact and sets the initial Terraform state.
func (r *ContactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var password types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if known(password) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Missing Password", "A password is required to create a contact.")
		return
	}

	body := contactRequest(&plan)
	body.Username = plan.Username.ValueString()
	body.Password = password.ValueString()

	id, err := contacts.Create(r.client, body)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Contact", fmt.Sprintf("Could not create contact %s: %s", plan.Username.ValueString(), err.Error()))
		return
	}

	// The contact exists from here on: record it before the read-back, so a
	// failed read leaves the contact in state instead of orphaning it.
	plan.ID = types.StringValue(strconv.Itoa(id))
	plan.Password = types.StringNull()
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	resp.Diagnostics.Append(r.readBack(&plan, id, "Created")...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ContactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Contact ID", fmt.Sprintf("The state holds %q, not a numeric contact id.", state.ID.ValueString()))
		return
	}

	contact, err := contacts.Get(r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Contact", fmt.Sprintf("Could not read contact %d: %s", id, err.Error()))
		return
	}
	if contact == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if contact.ID == 0 {
		contact.ID = id
	}

	state.fill(contact)
	if state.OnDestroy.IsNull() || state.OnDestroy.IsUnknown() {
		state.OnDestroy = types.StringValue(onDestroyRetain)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the contact to the plan.
func (r *ContactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Contact ID", fmt.Sprintf("The plan holds %q, not a numeric contact id.", plan.ID.ValueString()))
		return
	}

	if err := contacts.Update(r.client, id, contactRequest(&plan)); err != nil {
		resp.Diagnostics.AddError("Error Updating Contact", fmt.Sprintf("Could not update contact %d: %s", id, err.Error()))
		return
	}

	resp.Diagnostics.Append(r.readBack(&plan, id, "Updated")...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete does to the contact what `on_destroy` says.
func (r *ContactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Contact ID", fmt.Sprintf("The state holds %q, not a numeric contact id.", state.ID.ValueString()))
		return
	}

	// A state written without the attribute holds no value; treat it as retain.
	if state.OnDestroy.ValueString() != onDestroyDelete {
		resp.Diagnostics.AddWarning(
			"Contact Removed from Terraform State Only",
			fmt.Sprintf("Contact %d is no longer managed by Terraform but still exists at OpenProvider and can be imported again. "+
				"To delete it on destroy, set on_destroy = \"delete\" and apply before destroying.", id),
		)
		return
	}

	if err := contacts.Delete(r.client, id); err != nil {
		resp.Diagnostics.AddError("Error Deleting Contact", fmt.Sprintf("Could not delete contact %d: %s", id, err.Error()))
	}
}

// ImportState imports an existing contact by its numeric id.
func (r *ContactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil || id <= 0 {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected the numeric contact id, got %q.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	// Computed with a default; unknown after an import would plan a change.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("on_destroy"), onDestroyRetain)...)
}
