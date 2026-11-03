// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	path := fmt.Sprintf("/v1/domains/%d", id)
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

	var result DeleteDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Check if the API returned an error code (non-zero typically indicates error)
	if result.Code != 0 {
		return fmt.Errorf("delete operation failed with code %d", result.Code)
	}

	return nil
}
