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

// Contact represents the part of a contact this client reads.
type Contact struct {
	ID              int           `json:"id"`
	Username        string        `json:"username,omitempty"`
	Email           string        `json:"email,omitempty"`
	Role            string        `json:"role,omitempty"`
	RCPClientIPList *ClientIPList `json:"rcp_client_ip_list,omitempty"`
}
