// Package domains provides functionality for working with domains.
package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// AuthCode types. An `external` code transfers a domain to a different
// registrar, and an `internal` code transfers a domain to a different reseller
// account at Openprovider.
const (
	AuthCodeTypeExternal = "external"
	AuthCodeTypeInternal = "internal"
)

// AuthCode is the transfer authorization code of a domain.
type AuthCode struct {
	AuthCode string `json:"auth_code"`
	Type     string `json:"type"`
}

// AuthCodeResponse represents a response that carries an auth code.
type AuthCodeResponse struct {
	Code int      `json:"code"`
	Data AuthCode `json:"data"`
}

// ResetAuthCodeRequest represents a request to reset the auth code of a domain.
type ResetAuthCodeRequest struct {
	ID           int    `json:"id"`
	AuthCodeType string `json:"auth_code_type,omitempty"`
}

// GetAuthCode retrieves the auth code of a domain by ID from the Openprovider
// API. An empty `authCodeType` gives the `external` code.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/domains/{id}/authcode
func GetAuthCode(c *client.Client, id int, authCodeType string) (*AuthCode, error) {
	path := fmt.Sprintf("/v1beta/domains/%d/authcode", id)
	if authCodeType != "" {
		path += "?" + url.Values{"auth_code_type": {authCodeType}}.Encode()
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	return doAuthCode(c, req)
}

// ResetAuthCode replaces the auth code of a domain by ID with a new code via
// the Openprovider API, and returns the new code. The old code then no longer
// transfers the domain. An empty `authCodeType` resets the `external` code.
//
// Endpoint: POST https://api.openprovider.eu/v1beta/domains/{id}/authcode/reset
func ResetAuthCode(c *client.Client, id int, authCodeType string) (*AuthCode, error) {
	body, err := json.Marshal(&ResetAuthCodeRequest{ID: id, AuthCodeType: authCodeType})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/domains/%d/authcode/reset", id)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return doAuthCode(c, req)
}

func doAuthCode(c *client.Client, req *http.Request) (*AuthCode, error) {
	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result AuthCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
