package contacts

import (
	"fmt"

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
// Endpoint: GET https://api.openprovider.eu/v1/contacts/{id}
func Get(c *client.Client, id int) (*Contact, error) {
	var result GetContactResponse
	err := c.JSON("GET", fmt.Sprintf("/v1/contacts/%d", id), nil, &result)
	if client.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
