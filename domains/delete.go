// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/charpand/openprovider-go"
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
// Endpoint: DELETE https://api.openprovider.eu/v1beta/domains/{id}
func Delete(c *openprovider.Client, id int) error {
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

	return nil
}
