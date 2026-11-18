// Package nsgroups provides functionality for working with nameserver groups.
package nsgroups

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Delete deletes a nameserver group by name via the Openprovider API.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/dns/nameservers/groups/{ns_group}
func Delete(c *client.Client, name string) error {
	return c.JSON("DELETE", fmt.Sprintf("/v1/dns/nameservers/groups/%s", name), nil, nil)
}
