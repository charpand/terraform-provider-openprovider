package contacts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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

func send(c *client.Client, method, path string, payload any) (*http.Response, error) {
	var body *bytes.Buffer
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(b)
	}

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	} else {
		req, err = http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	}
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Create creates a contact and returns its id.
//
// Endpoint: POST https://api.openprovider.eu/v1beta/contacts
func Create(c *client.Client, req *Request) (int, error) {
	resp, err := send(c, "POST", "/v1beta/contacts", req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return 0, err
	}

	var result CreateContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if result.Data.ID == 0 {
		return 0, fmt.Errorf("the API did not return the id of the new contact")
	}
	return result.Data.ID, nil
}

// Update changes the fields that are set in req on an existing contact.
//
// Endpoint: PUT https://api.openprovider.eu/v1beta/contacts/{id}
func Update(c *client.Client, id int, req *Request) error {
	resp, err := send(c, "PUT", fmt.Sprintf("/v1beta/contacts/%d", id), req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	return err
}

// Delete deletes a contact. A contact that is already gone is not an error.
//
// Endpoint: DELETE https://api.openprovider.eu/v1beta/contacts/{id}
func Delete(c *client.Client, id int) error {
	resp, err := send(c, "DELETE", fmt.Sprintf("/v1beta/contacts/%d", id), nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}
