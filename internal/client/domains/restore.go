package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

type restoreDomainName struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

type restoreRequest struct {
	ID     int               `json:"id"`
	Domain restoreDomainName `json:"domain"`
}

type restoreResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

// Restore restores a domain that has expired, where the registry and the API
// allow it. It returns the status that the API answers with.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains/{id}/restore
func Restore(c *client.Client, id int, name, extension string) (string, error) {
	body, err := json.Marshal(&restoreRequest{
		ID:     id,
		Domain: restoreDomainName{Name: name, Extension: extension},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/domains/%d/restore", c.BaseURL, id), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return "", err
	}

	var result restoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// The API can refuse the restore with a non-zero code in a 2xx response.
	if result.Code != 0 {
		return "", fmt.Errorf("restore failed with code %d: %s", result.Code, result.Desc)
	}

	return result.Data.Status, nil
}
