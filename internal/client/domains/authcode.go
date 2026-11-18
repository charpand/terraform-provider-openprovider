// Package domains provides functionality for working with domains.
package domains

import (
	"fmt"
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
// Endpoint: GET https://api.openprovider.eu/v1/domains/{id}/authcode
func GetAuthCode(c *client.Client, id int, authCodeType string) (*AuthCode, error) {
	path := fmt.Sprintf("/v1/domains/%d/authcode", id)
	if authCodeType != "" {
		path += "?" + url.Values{"auth_code_type": {authCodeType}}.Encode()
	}

	var result AuthCodeResponse
	if err := c.JSON("GET", path, nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// ResetAuthCode replaces the auth code of a domain by ID with a new code via
// the Openprovider API, and returns the new code. The old code then no longer
// transfers the domain. An empty `authCodeType` resets the `external` code.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains/{id}/authcode/reset
func ResetAuthCode(c *client.Client, id int, authCodeType string) (*AuthCode, error) {
	var result AuthCodeResponse
	err := c.JSON("POST", fmt.Sprintf("/v1/domains/%d/authcode/reset", id), &ResetAuthCodeRequest{ID: id, AuthCodeType: authCodeType}, &result)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
