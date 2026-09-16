// Package nameservers provides functionality for working with glue records.
package nameservers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Create registers a glue record.
//
// Endpoint: POST /v1beta/dns/nameservers
func Create(c *client.Client, ns *Nameserver) error {
	body, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1beta/dns/nameservers", c.BaseURL), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = do(c, req)
	return err
}

// Update changes the addresses published for an existing glue record. The host
// name is the key, so it is passed in the path and cannot be changed here.
//
// Endpoint: PUT /v1beta/dns/nameservers/{name}
func Update(c *client.Client, name string, ns *Nameserver) error {
	body, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1beta/dns/nameservers/%s", c.BaseURL, name), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = do(c, req)
	return err
}

// Delete removes a glue record. A record that is already gone is not an error,
// because that is the state the caller asked for.
//
// Endpoint: DELETE /v1beta/dns/nameservers/{name}
func Delete(c *client.Client, name string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1beta/dns/nameservers/%s", c.BaseURL, name), nil)
	if err != nil {
		return err
	}

	_, err = do(c, req)
	if err == ErrNotFound {
		return nil
	}
	return err
}
