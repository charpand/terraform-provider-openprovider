// Package customers provides functionality for working with customers.
package customers

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Delete deletes a customer via the Openprovider API.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/customers/{handle}
func Delete(c *client.Client, handle string) error {
	return c.JSON("DELETE", fmt.Sprintf("/v1/customers/%s", handle), nil, nil)
}
