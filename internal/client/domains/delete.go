// Package domains provides functionality for working with domains.
package domains

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// DeleteDomainResponse represents a response for deleting a domain.
type DeleteDomainResponse struct {
	Code int `json:"code"`
	Data struct {
		Success bool `json:"success"`
	} `json:"data"`
}

// Delete deletes a domain by ID from the Openprovider API.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/domains/{id}
func Delete(c *client.Client, id int) error {
	var result DeleteDomainResponse
	if err := c.JSON("DELETE", fmt.Sprintf("/v1/domains/%d", id), nil, &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("delete operation failed with code %d", result.Code)
	}

	return nil
}
