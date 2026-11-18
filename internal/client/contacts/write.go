package contacts

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CreateContactResponse represents a response for creating a contact. The
// token the API returns with the id is not kept.
type CreateContactResponse struct {
	Code int `json:"code"`
	Data struct {
		ID int `json:"id"`
	} `json:"data"`
}

// Create creates a contact and returns its id.
//
// Endpoint: POST https://api.openprovider.eu/v1/contacts
func Create(c *client.Client, req *Request) (int, error) {
	var result CreateContactResponse
	if err := c.JSON("POST", "/v1/contacts", req, &result); err != nil {
		return 0, err
	}
	if result.Data.ID == 0 {
		return 0, fmt.Errorf("the API did not return the id of the new contact")
	}
	return result.Data.ID, nil
}

// Update changes the fields that are set in req on an existing contact.
//
// Endpoint: PUT https://api.openprovider.eu/v1/contacts/{id}
func Update(c *client.Client, id int, req *Request) error {
	return c.JSON("PUT", fmt.Sprintf("/v1/contacts/%d", id), req, nil)
}

// Delete deletes a contact. A contact that is already gone is not an error.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/contacts/{id}
func Delete(c *client.Client, id int) error {
	err := c.JSON("DELETE", fmt.Sprintf("/v1/contacts/%d", id), nil, nil)
	if client.IsNotFound(err) {
		return nil
	}
	return err
}
