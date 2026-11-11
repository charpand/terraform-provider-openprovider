package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

type restartLastOperationName struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

type restartLastOperationRequest struct {
	ID       int                      `json:"id"`
	Domain   restartLastOperationName `json:"domain"`
	AuthCode string                   `json:"auth_code,omitempty"`
}

type restartLastOperationResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Success *bool `json:"success"`
	} `json:"data"`
}

// RestartLastOperation sends the last operation of a domain to the registry
// again, without a new order. This operation is not always a transfer, so
// the caller must first make sure that the domain is in the expected status.
// An empty `authCode` sends no auth code.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains/{id}/last-operation/restart
func RestartLastOperation(c *client.Client, id int, name, extension, authCode string) error {
	body, err := json.Marshal(&restartLastOperationRequest{
		ID:       id,
		Domain:   restartLastOperationName{Name: name, Extension: extension},
		AuthCode: authCode,
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/domains/%d/last-operation/restart", id)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return err
	}

	var result restartLastOperationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// The API can refuse the operation with a non-zero code in a 2xx response.
	if result.Code != 0 {
		return fmt.Errorf("restart operation failed with code %d: %s", result.Code, result.Desc)
	}
	if result.Data.Success != nil && !*result.Data.Success {
		return fmt.Errorf("restart operation was not successful: %s", result.Desc)
	}

	return nil
}
