// Package domains provides functionality for working with domains.
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
// Endpoint: DELETE https://api.eu/v1beta/domains/{id}
func Delete(c *Client, id int) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1beta/domains/%d", c.BaseURL, id), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
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
