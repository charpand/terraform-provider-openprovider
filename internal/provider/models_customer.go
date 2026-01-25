// Package provider implements the Terraform provider for OpenProvider.
package provider

import (
	"github.com/charpand/terraform-provider-openprovider/internal/client/customers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AddressModel represents a customer address in Terraform state.
type AddressModel struct {
	City    types.String `tfsdk:"city"`
	Country types.String `tfsdk:"country"`
	Number  types.String `tfsdk:"number"`
	State   types.String `tfsdk:"state"`
	Street  types.String `tfsdk:"street"`
	Suffix  types.String `tfsdk:"suffix"`
	Zipcode types.String `tfsdk:"zipcode"`
}

// NameModel represents a customer name in Terraform state.
type NameModel struct {
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Initials  types.String `tfsdk:"initials"`
	Prefix    types.String `tfsdk:"prefix"`
}

// PhoneModel represents a customer phone in Terraform state.
type PhoneModel struct {
	AreaCode    types.String `tfsdk:"area_code"`
	CountryCode types.String `tfsdk:"country_code"`
	Number      types.String `tfsdk:"number"`
}

// CustomerModel represents the Terraform state model for a customer.
type CustomerModel struct {
	ID          types.String `tfsdk:"id"`
	Handle      types.String `tfsdk:"handle"`
	CompanyName types.String `tfsdk:"company_name"`
	Email       types.String `tfsdk:"email"`
	Locale      types.String `tfsdk:"locale"`
	Comments    types.String `tfsdk:"comments"`
	Phone       *PhoneModel  `tfsdk:"phone"`
	Address     *AddressModel `tfsdk:"address"`
	Name        *NameModel    `tfsdk:"name"`
}

// mapCustomerToModel converts a customer API response to a CustomerModel.
func mapCustomerToModel(customer *customers.Customer) *CustomerModel {
if customer == nil {
return nil
}

model := &CustomerModel{
Handle: types.StringValue(customer.Handle),
ID:     types.StringValue(customer.Handle),
Email:  types.StringValue(customer.Email),
}

if customer.CompanyName != "" {
model.CompanyName = types.StringValue(customer.CompanyName)
} else {
model.CompanyName = types.StringNull()
}

if customer.Locale != "" {
model.Locale = types.StringValue(customer.Locale)
} else {
model.Locale = types.StringNull()
}

if customer.Comments != "" {
model.Comments = types.StringValue(customer.Comments)
} else {
model.Comments = types.StringNull()
}

// Map phone
model.Phone = &PhoneModel{
CountryCode: types.StringValue(customer.Phone.CountryCode),
AreaCode:    types.StringValue(customer.Phone.AreaCode),
Number:      types.StringValue(customer.Phone.Number),
}

// Map address
model.Address = &AddressModel{
Street:  types.StringValue(customer.Address.Street),
City:    types.StringValue(customer.Address.City),
Country: types.StringValue(customer.Address.Country),
}
if customer.Address.Number != "" {
model.Address.Number = types.StringValue(customer.Address.Number)
} else {
model.Address.Number = types.StringNull()
}
if customer.Address.Suffix != "" {
model.Address.Suffix = types.StringValue(customer.Address.Suffix)
} else {
model.Address.Suffix = types.StringNull()
}
if customer.Address.State != "" {
model.Address.State = types.StringValue(customer.Address.State)
} else {
model.Address.State = types.StringNull()
}
if customer.Address.Zipcode != "" {
model.Address.Zipcode = types.StringValue(customer.Address.Zipcode)
} else {
model.Address.Zipcode = types.StringNull()
}

// Map name
model.Name = &NameModel{
FirstName: types.StringValue(customer.Name.FirstName),
LastName:  types.StringValue(customer.Name.LastName),
}
if customer.Name.Initials != "" {
model.Name.Initials = types.StringValue(customer.Name.Initials)
} else {
model.Name.Initials = types.StringNull()
}
if customer.Name.Prefix != "" {
model.Name.Prefix = types.StringValue(customer.Name.Prefix)
} else {
model.Name.Prefix = types.StringNull()
}

return model
}
