// Package customers provides functionality for working with customers.
package customers

import (
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Delete deletes a customer via the Openprovider API.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/customers/{handle}
func Delete(c *client.Client, handle string) error {
	path := fmt.Sprintf("/v1/customers/%s", handle)
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return nil
}
