package contacts

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// GetContactResponse represents a response for getting a contact.
type GetContactResponse struct {
	Code int     `json:"code"`
	Data Contact `json:"data"`
}

// Get retrieves a contact by id from the Openprovider API. It returns
// (nil, nil) for a contact that does not exist, so a caller can tell "gone"
// apart from a failed request.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/contacts/{id}
func Get(c *client.Client, id int) (*Contact, error) {
	path := fmt.Sprintf("/v1beta/contacts/%d", id)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		// `client.Client.Do` turns any non-2xx status into an error, so a
		// 404 is told apart here from the status alone.
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var result GetContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
