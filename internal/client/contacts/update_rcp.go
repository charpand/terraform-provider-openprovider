package contacts

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// rcpAllowList is the body of a control panel whitelist update. It carries
// the allow list and nothing else, so the deny list and every other field of
// the contact stay as they are.
type rcpAllowList struct {
	RCPClientIPList struct {
		Allow []string `json:"allow"`
	} `json:"rcp_client_ip_list"`
}

// SetRCPAllowList replaces the control panel IP whitelist of a contact. The
// list replaces the whole whitelist, and an empty list switches the
// whitelist off. A nil list is sent as an empty one.
//
// Endpoint: PUT https://api.openprovider.eu/v1/contacts/{id}
func SetRCPAllowList(c *client.Client, id int, allow []string) error {
	if allow == nil {
		allow = []string{}
	}
	var payload rcpAllowList
	payload.RCPClientIPList.Allow = allow

	return c.JSON("PUT", fmt.Sprintf("/v1/contacts/%d", id), payload, nil)
}
