// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// TemplateRecord is a record of a DNS template.
type TemplateRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"prio,omitempty"`
}

// Template is a set of DNS records that a new zone can start with.
type Template struct {
	ID                   int              `json:"id"`
	Name                 string           `json:"name"`
	IsSpamexpertsEnabled bool             `json:"is_spamexperts_enabled"`
	Records              []TemplateRecord `json:"records"`
}

// CreateTemplateRequest represents a request to create a DNS template.
type CreateTemplateRequest struct {
	Name                 string           `json:"name"`
	IsSpamexpertsEnabled bool             `json:"is_spamexperts_enabled,omitempty"`
	Records              []TemplateRecord `json:"records"`
}

type templateResponse struct {
	Code int      `json:"code"`
	Desc string   `json:"desc,omitempty"`
	Data Template `json:"data"`
}

type createTemplateResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		ID int `json:"id"`
	} `json:"data"`
}

// CreateTemplate creates a DNS template and returns its id.
//
// Endpoint: POST https://api.openprovider.eu/v1/dns/templates
func CreateTemplate(c *client.Client, req *CreateTemplateRequest) (int, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/dns/templates", c.BaseURL), bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return 0, err
	}

	var result createTemplateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if result.Code != 0 {
		return 0, fmt.Errorf("create template failed with code %d: %s", result.Code, result.Desc)
	}
	if result.Data.ID == 0 {
		return 0, fmt.Errorf("create template: the API answered without an id")
	}
	return result.Data.ID, nil
}

// GetTemplate retrieves a DNS template, with its records.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/templates/{id}
func GetTemplate(c *client.Client, id int) (*Template, error) {
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/dns/templates/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result templateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("get template failed with code %d: %s", result.Code, result.Desc)
	}
	return &result.Data, nil
}

// DeleteTemplate deletes a DNS template. A zone that was created from it
// keeps its records.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/dns/templates/{id}
func DeleteTemplate(c *client.Client, id int) error {
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/dns/templates/%d", c.BaseURL, id), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Desc string `json:"desc,omitempty"`
		Data struct {
			Success *bool `json:"success"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("delete template failed with code %d: %s", result.Code, result.Desc)
	}
	if result.Data.Success != nil && !*result.Data.Success {
		return fmt.Errorf("delete template was not successful: %s", result.Desc)
	}
	return nil
}
