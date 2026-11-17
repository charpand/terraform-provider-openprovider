package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
	body, err := json.Marshal(&renewRequest{
		ID:     id,
		Domain: renewDomainName{Name: name, Extension: extension},
		Period: period,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/domains/%d/renew", c.BaseURL, id), bytes.NewBuffer(body))
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

	var result renewResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// The API can refuse the renewal with a non-zero code in a 2xx response.
	if result.Code != 0 {
		return "", fmt.Errorf("renew failed with code %d: %s", result.Code, result.Desc)
	}

	return result.Data.Status, nil
}
