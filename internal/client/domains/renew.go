package domains

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

type renewDomainName struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

type renewRequest struct {
	ID     int             `json:"id"`
	Domain renewDomainName `json:"domain"`
	Period int             `json:"period"`
}

type renewResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

// Renew renews the domain with the id `id` for `period` years. It returns the
// status that the API answers with.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains/{id}/renew
func Renew(c *client.Client, id int, name, extension string, period int) (string, error) {
	var result renewResponse
	err := c.JSON("POST", fmt.Sprintf("/v1/domains/%d/renew", id), &renewRequest{
		ID:     id,
		Domain: renewDomainName{Name: name, Extension: extension},
		Period: period,
	}, &result)
	if err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("renew failed with code %d: %s", result.Code, result.Desc)
	}

	return result.Data.Status, nil
}
