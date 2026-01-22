// Package client provides functionality for working with domains.
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Domain represents a domain entity.
type Domain struct {
	ID             int    `json:"id"`
	ActiveDate     string `json:"active_date"`
	AdminHandle    string `json:"admin_handle"`
	AuthCode       string `json:"auth_code"`
	Autorenew      string `json:"autorenew"`
	BillingHandle  string `json:"billing_handle"`
	CanRenew       bool   `json:"can_renew"`
	CreationDate   string `json:"creation_date"`
	ExpirationDate string `json:"expiration_date"`
	IsAbusive      bool   `json:"is_abusive"`
	IsLocked       bool   `json:"is_locked"`
	LastChanged    string `json:"last_changed"`
	OrderDate      string `json:"order_date"`
	OwnerHandle    string `json:"owner_handle"`
	Status         string `json:"status"`
	TechHandle     string `json:"tech_handle"`
	Domain         struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
}

// ListDomainsResponse represents a response from the domains listing endpoint.
type ListDomainsResponse struct {
	Code int `json:"code"`
	Data struct {
		Results []Domain `json:"results"`
		Total   int      `json:"total"`
	} `json:"data"`
}

// List retrieves a list of domains from the Openprovider API.
func List(c *Client) ([]Domain, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/domains", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var results ListDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results.Data.Results, nil
}
