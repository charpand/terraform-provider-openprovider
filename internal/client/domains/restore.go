package domains

import (
	"fmt"

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
	var result restoreResponse
	err := c.JSON("POST", fmt.Sprintf("/v1/domains/%d/restore", id), &restoreRequest{
		ID:     id,
		Domain: restoreDomainName{Name: name, Extension: extension},
	}, &result)
	if err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("restore failed with code %d: %s", result.Code, result.Desc)
	}

	return result.Data.Status, nil
}
