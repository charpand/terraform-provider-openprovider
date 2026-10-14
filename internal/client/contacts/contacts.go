// Package contacts provides functionality for working with contacts, the
// objects that carry a login to the control panel (RCP) and the API.
//
// A contact is not a customer: it lives under `/v1beta/contacts/{id}` with an
// integer id, where a customer lives under `/v1beta/customers/{handle}`.
package contacts

// ClientIPList is the access list a contact carries for the control panel.
type ClientIPList struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny,omitempty"`
}

// Name represents the name of a contact.
type Name struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Initials  string `json:"initials,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

// Phone represents the phone number of a contact.
type Phone struct {
	CountryCode string `json:"country_code,omitempty"`
	AreaCode    string `json:"area_code,omitempty"`
	Number      string `json:"subscriber_number,omitempty"`
}

// Address represents the address of a contact.
type Address struct {
	Street  string `json:"street,omitempty"`
	Number  string `json:"number,omitempty"`
	Suffix  string `json:"suffix,omitempty"`
	Zipcode string `json:"zipcode,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Country string `json:"country,omitempty"`
}

// Contact represents the part of a contact this client reads. Fields the
// client does not manage, such as the secret key, are left out on purpose.
type Contact struct {
	ID               int           `json:"id"`
	Username         string        `json:"username,omitempty"`
	Email            string        `json:"email,omitempty"`
	Role             string        `json:"role,omitempty"`
	CompanyName      string        `json:"company_name,omitempty"`
	Locale           string        `json:"locale,omitempty"`
	Comments         string        `json:"comments,omitempty"`
	VAT              string        `json:"vat,omitempty"`
	IsActive         bool          `json:"is_active"`
	APIAccessEnabled bool          `json:"api_access_enabled"`
	Name             Name          `json:"name"`
	Phone            Phone         `json:"phone"`
	Address          Address       `json:"address"`
	RCPClientIPList  *ClientIPList `json:"rcp_client_ip_list,omitempty"`
}

// Request is the body of a create or an update of a contact. A field that is
// left empty (or nil) is not sent, so an update changes only what is set.
// Username and Password are accepted by a create only.
type Request struct {
	Username         string   `json:"username,omitempty"`
	Password         string   `json:"password,omitempty"`
	Email            string   `json:"email,omitempty"`
	Role             string   `json:"role,omitempty"`
	CompanyName      string   `json:"company_name,omitempty"`
	Locale           string   `json:"locale,omitempty"`
	Comments         string   `json:"comments,omitempty"`
	VAT              string   `json:"vat,omitempty"`
	IsActive         *bool    `json:"is_active,omitempty"`
	APIAccessEnabled *bool    `json:"api_access_enabled,omitempty"`
	Name             *Name    `json:"name,omitempty"`
	Phone            *Phone   `json:"phone,omitempty"`
	Address          *Address `json:"address,omitempty"`
}
